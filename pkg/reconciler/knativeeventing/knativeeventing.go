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
	"fmt"

	mf "github.com/manifestival/manifestival"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clientset "knative.dev/operator/pkg/client/clientset/versioned"

	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	knereconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeeventing"
	"knative.dev/operator/pkg/reconciler/common"
	kec "knative.dev/operator/pkg/reconciler/knativeeventing/common"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	oldFinalizerName = "delete-knative-eventing-manifest"
)

// Reconciler implements controller.Reconciler for KnativeEventing resources.
type Reconciler struct {
	// kubeClientSet allows us to talk to the k8s for core APIs
	kubeClientSet kubernetes.Interface
	// kubeClientSet allows us to talk to the k8s for operator APIs
	operatorClientSet clientset.Interface
	// manifest is empty but with a valid client and logger
	manifest mf.Manifest
	// Platform-specific behavior to affect the transform
	platform common.Extension
}

// Check that our Reconciler implements controller.Reconciler
var _ knereconciler.Interface = (*Reconciler)(nil)
var _ knereconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind removes all resources after deletion of a KnativeEventing.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *eventingv1alpha1.KnativeEventing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	// List all KnativeEventings to determine if cluster-scoped resources should be deleted.
	kes, err := r.operatorClientSet.OperatorV1alpha1().KnativeEventings("").List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all KnativeEventings: %w", err)
	}

	for _, ke := range kes.Items {
		if ke.GetDeletionTimestamp().IsZero() {
			// Not deleting all KnativeEventings. Nothing to do here.
			return nil
		}
	}

	manifest, err := common.InstalledManifest(original)
	if err != nil {
		return err
	}
	manifest = r.manifest.Append(manifest)
	if err := r.transform(ctx, &manifest, original); err != nil {
		return err
	}
	logger.Info("Deleting cluster-scoped resources")
	return common.Uninstall(&manifest)
}

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, ke *eventingv1alpha1.KnativeEventing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	ke.Status.InitializeConditions()
	ke.Status.ObservedGeneration = ke.Generation

	logger.Infow("Reconciling KnativeEventing", "status", ke.Status)
	stages := []func(context.Context, *mf.Manifest, *eventingv1alpha1.KnativeEventing) error{
		r.transform,
		r.ensureFinalizerRemoval,
		r.install,
		r.checkDeployments,
		r.deleteObsoleteResources,
	}

	manifest, err := common.TargetManifest(ke)
	if err != nil {
		return err
	}
	manifest = r.manifest.Append(manifest)
	for _, stage := range stages {
		if err := stage(ctx, &manifest, ke); err != nil {
			return err
		}
	}
	logger.Infow("Reconcile stages complete", "status", ke.Status)
	return nil
}

// transform mutates the passed manifest to one with common and
// platform transforms, plus any extras passed in
func (r *Reconciler) transform(ctx context.Context, manifest *mf.Manifest, instance *eventingv1alpha1.KnativeEventing) error {
	logger := logging.FromContext(ctx)
	return common.Transform(ctx, manifest, instance, r.platform,
		kec.DefaultBrokerConfigMapTransform(instance, logger))
}

// ensureFinalizerRemoval ensures that the obsolete "delete-knative-eventing-manifest" is removed from the resource.
func (r *Reconciler) ensureFinalizerRemoval(_ context.Context, _ *mf.Manifest, instance *eventingv1alpha1.KnativeEventing) error {
	patch, err := common.FinalizerRemovalPatch(instance, oldFinalizerName)
	if err != nil {
		return fmt.Errorf("failed to construct the patch: %w", err)
	}
	if patch == nil {
		// Nothing to do here.
		return nil
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
	return common.Install(manifest, common.TargetVersion(ke), &ke.Status)
}

func (r *Reconciler) checkDeployments(ctx context.Context, manifest *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Checking deployments")
	return common.CheckDeployments(r.kubeClientSet, manifest, &ke.Status)
}

// Delete obsolete resources from previous versions
func (r *Reconciler) deleteObsoleteResources(ctx context.Context, manifest *mf.Manifest, instance *eventingv1alpha1.KnativeEventing) error {
	resources := []*unstructured.Unstructured{
		// Remove old resources from 0.12
		// https://github.com/knative/eventing-operator/issues/90
		// sources and controller are merged.
		// delete removed or renamed resources.
		common.NamespacedResource("v1", "ServiceAccount", instance.GetNamespace(), "eventing-source-controller"),
		common.ClusterScopedResource("rbac.authorization.k8s.io/v1", "ClusterRole", "knative-eventing-source-controller"),
		common.ClusterScopedResource("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "knative-eventing-source-controller"),
		common.ClusterScopedResource("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "eventing-source-controller"),
		common.ClusterScopedResource("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "eventing-source-controller-resolver"),
		// Remove the legacysinkbindings webhook at 0.13
		common.ClusterScopedResource("admissionregistration.k8s.io/v1beta1", "MutatingWebhookConfiguration", "legacysinkbindings.webhook.sources.knative.dev"),
		// Remove the knative-eventing-sources-namespaced-admin ClusterRole at 0.13
		common.ClusterScopedResource("rbac.authorization.k8s.io/v1", "ClusterRole", "knative-eventing-sources-namespaced-admin"),
		// Remove the apiserversources.sources.eventing.knative.dev CRD at 0.13
		common.ClusterScopedResource("apiextensions.k8s.io/v1beta1", "CustomResourceDefinition", "apiserversources.sources.eventing.knative.dev"),
		// Remove the containersources.sources.eventing.knative.dev CRD at 0.13
		common.ClusterScopedResource("apiextensions.k8s.io/v1beta1", "CustomResourceDefinition", "containersources.sources.eventing.knative.dev"),
		// Remove the cronjobsources.sources.eventing.knative.dev CRD at 0.13
		common.ClusterScopedResource("apiextensions.k8s.io/v1beta1", "CustomResourceDefinition", "cronjobsources.sources.eventing.knative.dev"),
		// Remove the sinkbindings.sources.eventing.knative.dev CRD at 0.13
		common.ClusterScopedResource("apiextensions.k8s.io/v1beta1", "CustomResourceDefinition", "sinkbindings.sources.eventing.knative.dev"),
		// Remove the deployment sources-controller at 0.13
		common.NamespacedResource("apps/v1", "Deployment", instance.GetNamespace(), "sources-controller"),
		// Remove the resources at at 0.14
		common.NamespacedResource("v1", "ServiceAccount", instance.GetNamespace(), "pingsource-jobrunner"),
		common.NamespacedResource("batch/v1", "Job", instance.GetNamespace(), "v0.14.0-upgrade"),
		common.ClusterScopedResource("rbac.authorization.k8s.io/v1", "ClusterRole", "knative-eventing-jobrunner"),
		common.ClusterScopedResource("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "pingsource-jobrunner"),
	}
	for _, r := range resources {
		if err := manifest.Client.Delete(r); err != nil {
			return err
		}
	}
	return nil
}
