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
	"knative.dev/operator/version"
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
	// config is the manifest of KnativeEventing
	config mf.Manifest
	// Platform-specific behavior to affect the transform
	platform common.Platforms
}

// Check that our Reconciler implements controller.Reconciler
var _ knereconciler.Interface = (*Reconciler)(nil)
var _ knereconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind removes all resources after deletion of a KnativeEventing.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *eventingv1alpha1.KnativeEventing) pkgreconciler.Event {
	// List all KnativeEventings to determine if cluster-scoped resources should be deleted.
	kes, err := r.operatorClientSet.OperatorV1alpha1().KnativeEventings("").List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all KnativeEventings: %w", err)
	}

	// Only delete cluster-scoped resources if all KnativeEventings are being deleted.
	allBeingDeleted := true
	for _, ke := range kes.Items {
		if ke.GetDeletionTimestamp().IsZero() {
			allBeingDeleted = false
			break
		}
	}

	if allBeingDeleted {
		manifest, err := r.transform(ctx, original)
		if err != nil {
			return fmt.Errorf("failed to transform manifest: %w", err)
		}
		return common.RemoveClusterScoped(ctx, &manifest)
	}

	return nil
}

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, ke *eventingv1alpha1.KnativeEventing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	ke.Status.InitializeConditions()

	logger.Infow("Reconciling KnativeEventing", "status", ke.Status)
	stages := []func(context.Context, *mf.Manifest, *eventingv1alpha1.KnativeEventing) error{
		r.ensureFinalizerRemoval,
		func(ctx context.Context, mf *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
			return common.Install(ctx, version.EventingVersion, mf, &ke.Status)
		},
		func(ctx context.Context, mf *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
			return common.CheckDeployments(ctx, r.kubeClientSet, mf, &ke.Status)
		},
		r.deleteObsoleteResources,
	}

	manifest, err := r.transform(ctx, ke)
	if err != nil {
		ke.Status.MarkInstallFailed(err.Error())
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

	transforms := common.Transforms(ctx, instance)
	transforms = append(transforms, kec.DefaultBrokerConfigMapTransform(instance, logger))
	transforms = append(transforms, platform...)
	return r.config.Transform(transforms...)
}

// ensureFinalizerRemoval ensures that the obsolete "delete-knative-eventing-manifest" is removed from the resource.
func (r *Reconciler) ensureFinalizerRemoval(_ context.Context, _ *mf.Manifest, instance *eventingv1alpha1.KnativeEventing) error {
	patch, err := common.FinalizerRemovalPatch(instance, oldFinalizerName)
	if err != nil {
		return fmt.Errorf("failed to generate finalizer patch: %w", err)
	}
	if patch == nil {
		// Nothing to do
		return nil
	}

	patcher := r.operatorClientSet.OperatorV1alpha1().KnativeEventings(instance.Namespace)
	if _, err := patcher.Patch(instance.Name, types.MergePatchType, patch); err != nil {
		return fmt.Errorf("failed to patch finalizer away: %w", err)
	}
	return nil
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
	}
	for _, r := range resources {
		if err := manifest.Client.Delete(r); err != nil {
			return err
		}
	}
	return nil
}
