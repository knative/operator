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

package knativeserving

import (
	"context"
	"encoding/json"
	"fmt"

	mf "github.com/manifestival/manifestival"
	clientset "knative.dev/operator/pkg/client/clientset/versioned"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	knsreconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeserving"
	"knative.dev/operator/pkg/reconciler/common"
	ksc "knative.dev/operator/pkg/reconciler/knativeserving/common"
	"knative.dev/operator/version"
	"knative.dev/pkg/logging"

	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	oldFinalizerName = "delete-knative-serving-manifest"
)

var (
	role        mf.Predicate = mf.Any(mf.ByKind("ClusterRole"), mf.ByKind("Role"))
	rolebinding mf.Predicate = mf.Any(mf.ByKind("ClusterRoleBinding"), mf.ByKind("RoleBinding"))
)

// Reconciler implements controller.Reconciler for Knativeserving resources.
type Reconciler struct {
	// kubeClientSet allows us to talk to the k8s for core APIs
	kubeClientSet kubernetes.Interface
	// operatorClientSet allows us to configure operator objects
	operatorClientSet clientset.Interface
	// config is the manifest of KnativeServing
	config mf.Manifest
	// Platform-specific behavior to affect the transform
	platform common.Platforms
}

// Check that our Reconciler implements controller.Reconciler
var _ knsreconciler.Interface = (*Reconciler)(nil)
var _ knsreconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind removes all resources after deletion of a KnativeServing.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *servingv1alpha1.KnativeServing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	// List all KnativeServings to determine if cluster-scoped resources should be deleted.
	kss, err := r.operatorClientSet.OperatorV1alpha1().KnativeServings("").List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all KnativeServings: %w", err)
	}

	// Only delete cluster-scoped resources if all KnativeServings are being deleted.
	allBeingDeleted := true
	for _, ks := range kss.Items {
		if ks.GetDeletionTimestamp().IsZero() {
			allBeingDeleted = false
			break
		}
	}

	if allBeingDeleted {
		manifest, err := r.transform(ctx, original)
		if err != nil {
			return fmt.Errorf("failed to transform manifest: %w", err)
		}

		logger.Info("Deleting cluster-scoped resources")
		var rbac = mf.Any(mf.ByKind("Role"), mf.ByKind("ClusterRole"), mf.ByKind("RoleBinding"), mf.ByKind("ClusterRoleBinding"))
		if err := manifest.Filter(mf.NoCRDs, mf.None(rbac)).Delete(); err != nil {
			return fmt.Errorf("failed to remove non-crd/non-rbac resources: %w", err)
		}
		// Delete Roles last, as they may be useful for human operators to clean up.
		if err := manifest.Filter(rbac).Delete(); err != nil {
			return fmt.Errorf("failed to remove rbac: %w", err)
		}

		logger.Info("Cluster-scoped resources deleted")
	}
	return nil
}

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, ks *servingv1alpha1.KnativeServing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	ks.Status.InitializeConditions()

	logger.Infow("Reconciling KnativeServing", "status", ks.Status)
	stages := []func(context.Context, *mf.Manifest, *servingv1alpha1.KnativeServing) error{
		r.ensureFinalizerRemoval,
		r.install,
		r.checkDeployments,
		r.deleteObsoleteResources,
	}

	manifest, err := r.transform(ctx, ks)
	if err != nil {
		ks.Status.MarkInstallFailed(err.Error())
		return err
	}

	for _, stage := range stages {
		if err := stage(ctx, &manifest, ks); err != nil {
			return err
		}
	}
	logger.Infow("Reconcile stages complete", "status", ks.Status)
	return nil
}

// Transform the resources
func (r *Reconciler) transform(ctx context.Context, instance *servingv1alpha1.KnativeServing) (mf.Manifest, error) {
	logger := logging.FromContext(ctx)

	logger.Debug("Transforming manifest")

	platform, err := r.platform.Transformers(r.kubeClientSet, logger)
	if err != nil {
		return mf.Manifest{}, err
	}
	standard := []mf.Transformer{
		mf.InjectOwner(instance),
		mf.InjectNamespace(instance.GetNamespace()),
		common.ConfigMapTransform(instance.Spec.Config, logger),
		common.ImageTransform(&instance.Spec.Registry, logger),
		common.ResourceRequirementsTransform(instance.Spec.Resources, logger),
		ksc.GatewayTransform(instance, logger),
		ksc.CustomCertsTransform(instance, logger),
		ksc.HighAvailabilityTransform(instance, logger),
	}
	transforms := append(standard, platform...)
	return r.config.Transform(transforms...)
}

// Apply the manifest resources
func (r *Reconciler) install(ctx context.Context, manifest *mf.Manifest, instance *servingv1alpha1.KnativeServing) error {
	logger := logging.FromContext(ctx)

	logger.Debug("Installing manifest")
	// The Operator needs a higher level of permissions if it 'bind's non-existent roles.
	// To avoid this, we strictly order the manifest application as (Cluster)Roles, then
	// (Cluster)RoleBindings, then the rest of the manifest.
	if err := manifest.Filter(role).Apply(); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return err
	}
	if err := manifest.Filter(rolebinding).Apply(); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return err
	}
	if err := manifest.Apply(); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return err
	}
	instance.Status.MarkInstallSucceeded()
	instance.Status.Version = version.ServingVersion
	return nil
}

// Check for all deployments available
func (r *Reconciler) checkDeployments(ctx context.Context, manifest *mf.Manifest, instance *servingv1alpha1.KnativeServing) error {
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
			instance.Status.MarkDeploymentsNotReady()
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if !available(deployment) {
			instance.Status.MarkDeploymentsNotReady()
			return nil
		}
	}
	instance.Status.MarkDeploymentsAvailable()
	return nil
}

// ensureFinalizerRemoval ensures that the obsolete "delete-knative-serving-manifest" is removed from the resource.
func (r *Reconciler) ensureFinalizerRemoval(_ context.Context, _ *mf.Manifest, instance *servingv1alpha1.KnativeServing) error {
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

	patcher := r.operatorClientSet.OperatorV1alpha1().KnativeServings(instance.Namespace)
	if _, err := patcher.Patch(instance.Name, types.MergePatchType, patch); err != nil {
		return fmt.Errorf("failed to patch finalizer away: %w", err)
	}
	return nil
}

// Delete obsolete resources from previous versions
func (r *Reconciler) deleteObsoleteResources(ctx context.Context, manifest *mf.Manifest, instance *servingv1alpha1.KnativeServing) error {
	// istio-system resources from 0.3
	resource := &unstructured.Unstructured{}
	resource.SetNamespace("istio-system")
	resource.SetName("knative-ingressgateway")
	resource.SetAPIVersion("v1")
	resource.SetKind("Service")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}
	resource.SetAPIVersion("apps/v1")
	resource.SetKind("Deployment")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}
	resource.SetAPIVersion("autoscaling/v1")
	resource.SetKind("HorizontalPodAutoscaler")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}
	// config-controller from 0.5
	resource.SetNamespace(instance.GetNamespace())
	resource.SetName("config-controller")
	resource.SetAPIVersion("v1")
	resource.SetKind("ConfigMap")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}
	return nil
}
