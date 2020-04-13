/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package knativeeventing

import (
	"context"
	"reflect"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	listers "knative.dev/operator/pkg/client/listers/operator/v1alpha1"
	"knative.dev/operator/pkg/reconciler"
	"knative.dev/operator/pkg/reconciler/knativeeventing/common"
	"knative.dev/operator/version"
	"knative.dev/pkg/controller"
)

const (
	finalizerName = "delete-knative-eventing-manifest"
)

var (
	platform    common.Platforms
	role        mf.Predicate = mf.Any(mf.ByKind("ClusterRole"), mf.ByKind("Role"))
	rolebinding mf.Predicate = mf.Any(mf.ByKind("ClusterRoleBinding"), mf.ByKind("RoleBinding"))
)

// Reconciler implements controller.Reconciler for Knativeeventing resources.
type Reconciler struct {
	*reconciler.Base
	// Listers index properties about resources
	knativeEventingLister listers.KnativeEventingLister
	config                mf.Manifest
	eventings             sets.String
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Knativeeventing resource
// with the current status of the resource.
func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		r.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	// Get the KnativeEventing resource with this namespace/name.
	original, err := r.knativeEventingLister.KnativeEventings(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		return nil
	} else if err != nil {
		r.Logger.Error(err, "Error getting KnativeEventings")
		return err
	}

	if original.GetDeletionTimestamp() != nil {
		if _, ok := r.eventings[key]; ok {
			delete(r.eventings, key)
		}
		return r.delete(original)
	}

	// Keep track of the number of Eventings in the cluster
	r.eventings.Insert(key)

	// Don't modify the informers copy.
	knativeEventing := original.DeepCopy()

	// Reconcile this copy of the KnativeEventing resource and then write back any status
	// updates regardless of whether the reconciliation errored out.
	reconcileErr := r.reconcile(ctx, knativeEventing)
	if equality.Semantic.DeepEqual(original.Status, knativeEventing.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if _, err = r.updateStatus(knativeEventing); err != nil {
		r.Logger.Warnw("Failed to update KnativeEventing status", zap.Error(err))
		r.Recorder.Eventf(knativeEventing, corev1.EventTypeWarning, "UpdateFailed",
			"Failed to update status for KnativeEventing %q: %v", knativeEventing.Name, err)
		return err
	}
	if reconcileErr != nil {
		r.Recorder.Event(knativeEventing, corev1.EventTypeWarning, "InternalError", reconcileErr.Error())
		return reconcileErr
	}
	return nil
}

func (r *Reconciler) reconcile(ctx context.Context, ke *eventingv1alpha1.KnativeEventing) error {
	reqLogger := r.Logger.With(zap.String("Request.Namespace", ke.Namespace)).With("Request.Name", ke.Name)
	reqLogger.Infow("Reconciling KnativeEventing", "status", ke.Status)

	stages := []func(*mf.Manifest, *eventingv1alpha1.KnativeEventing) error{
		r.ensureFinalizer,
		r.initStatus,
		r.install,
		r.checkDeployments,
		r.deleteObsoleteResources,
	}

	manifest, err := r.transform(ke)
	if err != nil {
		ke.Status.MarkEventingFailed("Manifest Installation", err.Error())
		return err
	}

	for _, stage := range stages {
		if err := stage(&manifest, ke); err != nil {
			return err
		}
	}
	reqLogger.Infow("Reconcile stages complete", "status", ke.Status)
	return nil
}

func (r *Reconciler) initStatus(_ *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	r.Logger.Debug("Initializing status")
	if len(ke.Status.Conditions) == 0 {
		ke.Status.InitializeConditions()
		if _, err := r.updateStatus(ke); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) transform(instance *eventingv1alpha1.KnativeEventing) (mf.Manifest, error) {
	r.Logger.Debug("Transforming manifest")
	transforms, err := platform.Transformers(r.KubeClientSet, instance, r.Logger)
	if err != nil {
		return mf.Manifest{}, err
	}
	return r.config.Transform(transforms...)
}

func (r *Reconciler) install(manifest *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	r.Logger.Debug("Installing manifest")
	defer r.updateStatus(ke)
	// The Operator needs a higher level of permissions if it 'bind's non-existent roles.
	// To avoid this, we strictly order the manifest application as (Cluster)Roles, then
	// (Cluster)RoleBindings, then the rest of the manifest.
	if err := manifest.Filter(role).Apply(); err != nil {
		ke.Status.MarkEventingFailed("Manifest Installation", err.Error())
		return err
	}
	if err := manifest.Filter(rolebinding).Apply(); err != nil {
		ke.Status.MarkEventingFailed("Manifest Installation", err.Error())
		return err
	}
	if err := manifest.Filter(mf.None(role, rolebinding)).Apply(); err != nil {
		ke.Status.MarkEventingFailed("Manifest Installation", err.Error())
		return err
	}
	ke.Status.Version = version.EventingVersion
	ke.Status.MarkInstallationReady()
	return nil
}

func (r *Reconciler) checkDeployments(manifest *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	r.Logger.Debug("Checking deployments")
	defer r.updateStatus(ke)
	available := func(d *appsv1.Deployment) bool {
		for _, c := range d.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	}
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		deployment, err := r.KubeClientSet.AppsV1().Deployments(u.GetNamespace()).Get(u.GetName(), metav1.GetOptions{})
		if err != nil {
			ke.Status.MarkEventingNotReady("Deployment check", err.Error())
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if !available(deployment) {
			ke.Status.MarkEventingNotReady("Deployment check", "The deployment is not available.")
			return nil
		}
	}
	ke.Status.MarkEventingReady()
	return nil
}

func (r *Reconciler) updateStatus(desired *eventingv1alpha1.KnativeEventing) (*eventingv1alpha1.KnativeEventing, error) {
	ke, err := r.KnativeEventingClientSet.OperatorV1alpha1().KnativeEventings(desired.Namespace).Get(desired.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// If there's nothing to update, just return.
	if reflect.DeepEqual(ke.Status, desired.Status) {
		return ke, nil
	}
	// Don't modify the informers copy
	existing := ke.DeepCopy()
	existing.Status = desired.Status
	return r.KnativeEventingClientSet.OperatorV1alpha1().KnativeEventings(desired.Namespace).UpdateStatus(existing)
}

// ensureFinalizer attaches a "delete manifest" finalizer to the instance
func (r *Reconciler) ensureFinalizer(manifest *mf.Manifest, instance *eventingv1alpha1.KnativeEventing) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName {
			return nil
		}
	}
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName))
	instance, err := r.KnativeEventingClientSet.OperatorV1alpha1().KnativeEventings(instance.Namespace).Update(instance)
	return err
}

// delete all the resources in the release manifest
func (r *Reconciler) delete(instance *eventingv1alpha1.KnativeEventing) error {
	if len(instance.GetFinalizers()) == 0 || instance.GetFinalizers()[0] != finalizerName {
		return nil
	}
	r.Logger.Info("Deleting resources")
	var RBAC = mf.Any(role, rolebinding)
	if len(r.eventings) == 0 {
		// delete the deployments first
		if err := r.config.Filter(mf.ByKind("Deployment")).Delete(); err != nil {
			r.Logger.Warn(err, "Error deleting deployments")
			return err
		}
		if err := r.config.Filter(mf.NoCRDs, mf.None(RBAC)).Delete(); err != nil {
			r.Logger.Warn(err, "Error deleting resources")
			return err
		}
		// Delete Roles last, as they may be useful for human operators to clean up.
		if err := r.config.Filter(RBAC).Delete(); err != nil {
			r.Logger.Warn(err, "Error deleting RBAC resources")
			return err
		}

		r.Logger.Info("Resources are deleted")
	}
	// The deletionTimestamp might've changed. Fetch the resource again.
	refetched, err := r.knativeEventingLister.KnativeEventings(instance.Namespace).Get(instance.Name)
	if err != nil {
		return err
	}
	refetched.SetFinalizers(refetched.GetFinalizers()[1:])
	_, err = r.KnativeEventingClientSet.OperatorV1alpha1().KnativeEventings(refetched.Namespace).Update(refetched)
	return err
}

// Delete obsolete resources from previous versions
func (r *Reconciler) deleteObsoleteResources(manifest *mf.Manifest, instance *eventingv1alpha1.KnativeEventing) error {
	resource := &unstructured.Unstructured{}
	resource.SetNamespace(instance.GetNamespace())

	// Remove old resources from 0.12
	// https://github.com/knative/eventing-operator/issues/90
	// sources and controller are merged.
	// delete removed or renamed resources.
	resource.SetAPIVersion("v1")
	resource.SetKind("ServiceAccount")
	resource.SetName("eventing-source-controller")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	resource.SetAPIVersion("rbac.authorization.k8s.io/v1")
	resource.SetKind("ClusterRole")
	resource.SetName("knative-eventing-source-controller")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	resource.SetAPIVersion("rbac.authorization.k8s.io/v1")
	resource.SetKind("ClusterRoleBinding")
	resource.SetName("eventing-source-controller")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	resource.SetAPIVersion("rbac.authorization.k8s.io/v1")
	resource.SetKind("ClusterRoleBinding")
	resource.SetName("eventing-source-controller-resolver")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	// Remove the legacysinkbindings webhook at 0.13
	resource.SetAPIVersion("admissionregistration.k8s.io/v1beta1")
	resource.SetKind("MutatingWebhookConfiguration")
	resource.SetName("legacysinkbindings.webhook.sources.knative.dev")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	// Remove the knative-eventing-sources-namespaced-admin ClusterRole at 0.13
	resource.SetAPIVersion("rbac.authorization.k8s.io/v1")
	resource.SetKind("ClusterRole")
	resource.SetName("knative-eventing-sources-namespaced-admin")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	// Remove the apiserversources.sources.eventing.knative.dev CRD at 0.13
	resource.SetAPIVersion("apiextensions.k8s.io/v1beta1")
	resource.SetKind("CustomResourceDefinition")
	resource.SetName("apiserversources.sources.eventing.knative.dev")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	// Remove the containersources.sources.eventing.knative.dev CRD at 0.13
	resource.SetAPIVersion("apiextensions.k8s.io/v1beta1")
	resource.SetKind("CustomResourceDefinition")
	resource.SetName("containersources.sources.eventing.knative.dev")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	// Remove the cronjobsources.sources.eventing.knative.dev CRD at 0.13
	resource.SetAPIVersion("apiextensions.k8s.io/v1beta1")
	resource.SetKind("CustomResourceDefinition")
	resource.SetName("cronjobsources.sources.eventing.knative.dev")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	// Remove the sinkbindings.sources.eventing.knative.dev CRD at 0.13
	resource.SetAPIVersion("apiextensions.k8s.io/v1beta1")
	resource.SetKind("CustomResourceDefinition")
	resource.SetName("sinkbindings.sources.eventing.knative.dev")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	// Remove the deployment sources-controller at 0.13
	resource.SetAPIVersion("apps/v1")
	resource.SetKind("deployment")
	resource.SetName("sources-controller")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}
	return nil
}
