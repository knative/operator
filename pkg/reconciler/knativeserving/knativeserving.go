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
	"fmt"

	mf "github.com/manifestival/manifestival"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	clientset "knative.dev/operator/pkg/client/clientset/versioned"
	knsreconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeserving"
	"knative.dev/operator/pkg/reconciler/common"
	ksc "knative.dev/operator/pkg/reconciler/knativeserving/common"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	oldFinalizerName = "delete-knative-serving-manifest"
)

// Reconciler implements controller.Reconciler for Knativeserving resources.
type Reconciler struct {
	// kubeClientSet allows us to talk to the k8s for core APIs
	kubeClientSet kubernetes.Interface
	// operatorClientSet allows us to configure operator objects
	operatorClientSet clientset.Interface
	// manifest is empty but with a valid client and logger
	manifest mf.Manifest
	// Platform-specific behavior to affect the transform
	platform common.Extension
}

// Check that our Reconciler implements controller.Reconciler
var _ knsreconciler.Interface = (*Reconciler)(nil)
var _ knsreconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind removes all resources after deletion of a KnativeServing.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *v1alpha1.KnativeServing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	// List all KnativeServings to determine if cluster-scoped resources should be deleted.
	kss, err := r.operatorClientSet.OperatorV1alpha1().KnativeServings("").List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all KnativeServings: %w", err)
	}

	for _, ks := range kss.Items {
		if ks.GetDeletionTimestamp().IsZero() {
			// Not deleting all KnativeServings. Nothing to do here.
			return nil
		}
	}

	// TODO: which manifest should we delete? both?
	version := ""
	if original.GetStatus().IsReady() {
		version = original.Status.Version
	} else {
		version = common.TargetRelease(original)
	}

	manifest, err := common.FetchManifest(common.ManifestPath(version, common.PathElement(original)))
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
func (r *Reconciler) ReconcileKind(ctx context.Context, ks *v1alpha1.KnativeServing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	ks.Status.InitializeConditions()
	ks.Status.ObservedGeneration = ks.Generation

	logger.Infow("Reconciling KnativeServing", "status", ks.Status)
	stages := []func(context.Context, *mf.Manifest, v1alpha1.KComponent) error{
		r.create,
		// r.ingress -- should append to the one passed to it
		r.transform,
		r.ensureFinalizerRemoval,
		r.install,
		r.checkDeployments,
		r.deleteObsoleteResources,
		r.updateVersion,
	}

	// Appending nothing effectively results in a deep-copy clone
	manifest := r.manifest.Append()
	for _, stage := range stages {
		if err := stage(ctx, &manifest, ks); err != nil {
			return err
		}
	}
	logger.Infow("Reconcile stages complete", "status", ks.Status)
	return nil
}

// create mutates the passed manifest, appending to it one appropriate
// for the specified version in the instance
func (r *Reconciler) create(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	path := common.ManifestPath(common.TargetRelease(instance), common.PathElement(instance))
	m, err := common.FetchManifest(path)
	if err != nil {
		return err
	}
	*manifest = manifest.Append(m)
	return nil
}

// transform mutates the passed manifest to one with common and
// platform transforms, plus any extras passed in
func (r *Reconciler) transform(ctx context.Context, manifest *mf.Manifest, component v1alpha1.KComponent) error {
	logger := logging.FromContext(ctx)
	instance := component.(*v1alpha1.KnativeServing)
	return common.Transform(ctx, manifest, instance, r.platform,
		ksc.GatewayTransform(instance, logger),
		ksc.CustomCertsTransform(instance, logger),
		ksc.HighAvailabilityTransform(instance, logger),
		ksc.AggregationRuleTransform(manifest.Client))
}

// Apply the manifest resources
func (r *Reconciler) install(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Installing manifest")
	return common.Install(manifest, instance.GetStatus())
}

// Check for all deployments available
func (r *Reconciler) checkDeployments(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Checking deployments")
	return common.CheckDeployments(r.kubeClientSet, manifest, instance.GetStatus())
}

// ensureFinalizerRemoval ensures that the obsolete "delete-knative-serving-manifest" is removed from the resource.
func (r *Reconciler) ensureFinalizerRemoval(_ context.Context, _ *mf.Manifest, instance v1alpha1.KComponent) error {
	patch, err := common.FinalizerRemovalPatch(instance, oldFinalizerName)
	if err != nil {
		return fmt.Errorf("failed to construct the patch: %w", err)
	}
	if patch == nil {
		// Nothing to do here.
		return nil
	}

	patcher := r.operatorClientSet.OperatorV1alpha1().KnativeServings(instance.GetNamespace())
	if _, err := patcher.Patch(instance.GetName(), types.MergePatchType, patch); err != nil {
		return fmt.Errorf("failed to patch finalizer away: %w", err)
	}
	return nil
}

// Delete obsolete resources from previous versions, i.e. remove the
// resources that do not exist in the passed target manifest
func (r *Reconciler) deleteObsoleteResources(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	if instance.GetStatus().GetVersion() == "" {
		return nil
	}
	if common.TargetRelease(instance) == instance.GetStatus().GetVersion() {
		return nil
	}
	logger := logging.FromContext(ctx)
	m, err := common.FetchManifest(common.ManifestPath(instance.GetStatus().GetVersion(), common.PathElement(instance)))
	if err != nil {
		logger.Error(err, "Unable to fetch previous manifest; some obsolete resources may remain")
		return nil
	}
	m = r.manifest.Append(m)
	if err := r.transform(ctx, &m, instance); err != nil {
		return err
	}
	return m.Filter(mf.None(mf.In(*manifest))).Delete()
}

// updateVersion sets the status version if all conditions are satisfied
func (r *Reconciler) updateVersion(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	if instance.GetStatus().IsReady() {
		instance.GetStatus().SetVersion(common.TargetRelease(instance))
	}
	return nil
}
