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
	// manifests is the map of Knative Serving manifests
	manifests map[string]mf.Manifest
	// mfClient is the client needed for manifestival.
	mfClient mf.Client
	// Platform-specific behavior to affect the transform
	platform common.Platforms
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

	manifest, err := r.getCurrentManifest(ctx, original)
	if err != nil {
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
	// Get the target Serving Manifest to be installed
	targetManifest, err := r.getTargetManifest(ctx, ke)
	if err != nil {
		ke.Status.MarkInstallFailed(err.Error())
		return err
	}

	// Get the current Manifest, which has been installed.
	currentManifest, err := r.getCurrentManifest(ctx, ke)
	if err != nil {
		return err
	}

	stages := []func(context.Context, *mf.Manifest, *eventingv1alpha1.KnativeEventing) error{
		r.ensureFinalizerRemoval,
		r.install,
		r.checkDeployments,
	}

	for _, stage := range stages {
		if err := stage(ctx, &targetManifest, ke); err != nil {
			return err
		}
	}

	// Remove the resources that do not exist in the new Eventing manifest
	if err := currentManifest.Filter(mf.None(mf.In(targetManifest))).Delete(); err != nil {
		return err
	}
	logger.Infow("Reconcile stages complete", "status", ke.Status)
	return nil
}

func (r *Reconciler) transform(ctx context.Context, instance *eventingv1alpha1.KnativeEventing) ([]mf.Transformer, error) {
	logger := logging.FromContext(ctx)
	logger.Debug("Transforming manifest")

	platform, err := r.platform.Transformers(r.kubeClientSet, logger)
	if err != nil {
		return []mf.Transformer{}, err
	}

	transformers := common.Transformers(ctx, instance)
	transformers = append(transformers, kec.DefaultBrokerConfigMapTransform(instance, logger))
	transformers = append(transformers, platform...)
	return transformers, nil
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

	version := ke.Spec.GetVersion()
	var err error = nil
	if version == "" {
		version = ke.Status.GetVersion()
	}

	if version == "" {
		version, err = common.GetLatestRelease("knative-serving")
	}

	if err != nil {
		return err
	}

	return common.Install(manifest, version, &ke.Status)
}

func (r *Reconciler) checkDeployments(ctx context.Context, manifest *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Checking deployments")
	return common.CheckDeployments(r.kubeClientSet, manifest, &ke.Status)
}

// getTargetManifest returns the manifest to be installed
func (r *Reconciler) getTargetManifest(ctx context.Context, instance *eventingv1alpha1.KnativeEventing) (mf.Manifest, error) {
	if instance.Spec.GetVersion() != "" {
		// If the version is set in spec of the CR, pick the version from the spec of the CR
		return r.tranformManifest(ctx, instance.Spec.GetVersion(), instance)
	}

	version := instance.Status.GetVersion()
	if version == "" {
		return r.getLatestManifest(ctx, instance)
	}

	ver, err := common.GetEarliestSupportedRelease("knative-eventing")
	if err == nil && version < ver {
		// If the version of the existing Knative eventing deployment is prior to the earliest supported version,
		// we need to pick the earliest supported version.
		version = ver
	}

	return r.tranformManifest(ctx, version, instance)
}

// getCurrentManifest returns the manifest which has been installed
func (r *Reconciler) getCurrentManifest(ctx context.Context, instance *eventingv1alpha1.KnativeEventing) (mf.Manifest, error) {
	if instance.Status.GetVersion() != "" {
		// If the version is set in the status of the CR, pick the version from the status of the CR
		return r.tranformManifest(ctx, instance.Status.GetVersion(), instance)
	}

	return r.getLatestManifest(ctx, instance)
}

// getLatestManifest returns the manifest of the latest version locally available
func (r *Reconciler) getLatestManifest(ctx context.Context, instance *eventingv1alpha1.KnativeEventing) (mf.Manifest, error) {
	// The version is set to the default version
	version, err := common.GetLatestRelease("knative-eventing")
	if err != nil {
		return mf.Manifest{}, err
	}

	return r.tranformManifest(ctx, version, instance)
}

// transformManifest tranforms the manifest by providing the version number and the Knative Eventing CR
func (r *Reconciler) tranformManifest(ctx context.Context, version string,
	instance *eventingv1alpha1.KnativeEventing) (mf.Manifest, error) {
	manifest, found := r.manifests[version]
	var err error = nil
	if !found {
		manifest, err = common.RetrieveManifest(ctx, version, "eventing", r.mfClient, yamlList)
		if err != nil {
			return manifest, err
		}

		// Save the manifest in the map
		r.manifests[version] = manifest
	}

	// Create the transformer for Knative Eventing
	transformers, err := r.transform(ctx, instance)
	if err != nil {
		return mf.Manifest{}, err
	}

	// Transform the manifest
	manifestTransformed, err := manifest.Transform(transformers...)
	if err != nil {
		return mf.Manifest{}, err
	}

	return manifestTransformed, nil
}
