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
	"encoding/json"
	"fmt"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	clientset "knative.dev/operator/pkg/client/clientset/versioned"

	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	knereconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeeventing"
	"knative.dev/operator/pkg/reconciler/common"
	kec "knative.dev/operator/pkg/reconciler/knativeeventing/common"
	"knative.dev/operator/version"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	oldFinalizerName = "delete-knative-eventing-manifest"
)

var (
	role        mf.Predicate = mf.Any(mf.ByKind("ClusterRole"), mf.ByKind("Role"))
	rolebinding mf.Predicate = mf.Any(mf.ByKind("ClusterRoleBinding"), mf.ByKind("RoleBinding"))
)

// Reconciler implements controller.Reconciler for KnativeEventing resources.
type Reconciler struct {
	// kubeClientSet allows us to talk to the k8s for core APIs
	kubeClientSet kubernetes.Interface
	// kubeClientSet allows us to talk to the k8s for operator APIs
	operatorClientSet clientset.Interface
	config            mf.Manifest
	eventings         sets.String
	// Platform-specific behavior to affect the transform
	platform common.Platforms
}

// Check that our Reconciler implements controller.Reconciler
var _ knereconciler.Interface = (*Reconciler)(nil)
var _ knereconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind removes all resources after deletion of a KnativeEventing.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *eventingv1alpha1.KnativeEventing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	r.eventings.Delete(key(original))

	if len(r.eventings) == 0 {
		logger.Info("Deleting resources")
		RBAC := mf.Any(role, rolebinding)

		// delete the deployments first
		if err := r.config.Filter(mf.ByKind("Deployment")).Delete(); err != nil {
			return err
		}
		if err := r.config.Filter(mf.NoCRDs, mf.None(RBAC)).Delete(); err != nil {
			return err
		}
		// Delete Roles last, as they may be useful for human operators to clean up.
		if err := r.config.Filter(RBAC).Delete(); err != nil {
			return err
		}

		logger.Info("Resources are deleted")
	}

	return nil
}

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, ke *eventingv1alpha1.KnativeEventing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	ke.Status.InitializeConditions()

	// Keep track of the number of Eventings in the cluster
	r.eventings.Insert(key(ke))

	logger.Infow("Reconciling KnativeEventing", "status", ke.Status)
	stages := []func(context.Context, *mf.Manifest, *eventingv1alpha1.KnativeEventing) error{
		r.ensureFinalizerRemoval,
		r.install,
		r.checkDeployments,
		r.deleteObsoleteResources,
	}

	manifest, err := r.transform(ctx, ke)
	if err != nil {
		ke.Status.MarkEventingFailed("Manifest Installation", err.Error())
		return err
	}

	for _, stage := range stages {
		if err := stage(ctx, &manifest, ke); err != nil {
			return err
		}
	}
	logger.Infow("Reconcile stages complete", "status", ke.Status)
	return nil
}

func (r *Reconciler) transform(ctx context.Context, instance *eventingv1alpha1.KnativeEventing) (mf.Manifest, error) {
	logger := logging.FromContext(ctx)
	logger.Debug("Transforming manifest")

	platform, err := r.platform.Transformers(r.kubeClientSet, logger)
	if err != nil {
		return mf.Manifest{}, err
	}
	standard := []mf.Transformer{
		mf.InjectOwner(instance),
		mf.InjectNamespace(instance.GetNamespace()),
		common.ImageTransform(&instance.Spec.Registry, logger),
		common.ConfigMapTransform(instance.Spec.Config, logger),
		kec.DefaultBrokerConfigMapTransform(instance, logger),
	}
	transforms := append(standard, platform...)
	return r.config.Transform(transforms...)
}

// ensureFinalizerRemoval ensures that the obsolete "delete-knative-eventing-manifest" is removed from the resource.
func (r *Reconciler) ensureFinalizerRemoval(_ context.Context, _ *mf.Manifest, instance *eventingv1alpha1.KnativeEventing) error {
	finalizers := sets.NewString(instance.Finalizers...)

	if !finalizers.Has(oldFinalizerName) {
		// Nothing to do.
		return nil
	}

	// Remove the finalizer
	finalizers.Delete(oldFinalizerName)

	mergePatch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers":      finalizers.List(),
			"resourceVersion": instance.ResourceVersion,
		},
	}

	patch, err := json.Marshal(mergePatch)
	if err != nil {
		return fmt.Errorf("failed to construct finalizer patch: %w", err)
	}

	patcher := r.operatorClientSet.OperatorV1alpha1().KnativeEventings(instance.Namespace)
	if _, err := patcher.Patch(instance.Name, types.MergePatchType, patch); err != nil {
		return fmt.Errorf("failed to patch finalizer away: %w", err)
	}
	return nil
}

func (r *Reconciler) install(ctx context.Context, manifest *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Installing manifest")
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

func (r *Reconciler) checkDeployments(ctx context.Context, manifest *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Checking deployments")
	available := func(d *appsv1.Deployment) bool {
		for _, c := range d.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	}
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		deployment, err := r.kubeClientSet.AppsV1().Deployments(u.GetNamespace()).Get(u.GetName(), metav1.GetOptions{})
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

// Delete obsolete resources from previous versions
func (r *Reconciler) deleteObsoleteResources(ctx context.Context, manifest *mf.Manifest, instance *eventingv1alpha1.KnativeEventing) error {
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

func key(kne *eventingv1alpha1.KnativeEventing) string {
	return fmt.Sprintf("%s/%s", kne.Namespace, kne.Name)
}
