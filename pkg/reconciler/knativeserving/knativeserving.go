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

	"knative.dev/operator/pkg/reconciler/knativeserving/ingress"

	mf "github.com/manifestival/manifestival"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	clientset "knative.dev/operator/pkg/client/clientset/versioned"
	knsreconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeserving"
	"knative.dev/operator/pkg/reconciler/common"
	ksc "knative.dev/operator/pkg/reconciler/knativeserving/common"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler implements controller.Reconciler for Knativeserving resources.
type Reconciler struct {
	// kubeClientSet allows us to talk to the k8s for core APIs
	kubeClientSet kubernetes.Interface
	// operatorClientSet allows us to configure operator objects
	operatorClientSet clientset.Interface
	// manifest is empty, but with a valid client and logger. all
	// manifests are immutable, and any created during reconcile are
	// expected to be appended to this one, obviating the passing of
	// client & logger
	manifest mf.Manifest
	// Platform-specific behavior to affect the transform
	extension common.Extension
}

// Check that our Reconciler implements controller.Reconciler
var _ knsreconciler.Interface = (*Reconciler)(nil)
var _ knsreconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind removes all resources after deletion of a KnativeServing.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *v1alpha1.KnativeServing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	// List all KnativeServings to determine if cluster-scoped resources should be deleted.
	kss, err := r.operatorClientSet.OperatorV1alpha1().KnativeServings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all KnativeServings: %w", err)
	}

	for _, ks := range kss.Items {
		if ks.GetDeletionTimestamp().IsZero() {
			// Not deleting all KnativeServings. Nothing to do here.
			return nil
		}
	}

	if err := r.extension.Finalize(ctx, original); err != nil {
		logger.Error("Failed to finalize platform resources", err)
	}
	logger.Info("Deleting cluster-scoped resources")
	manifest, err := r.installed(ctx, original)
	if err != nil {
		logger.Error("Unable to fetch installed manifest; no cluster-scoped resources will be finalized", err)
		return nil
	}
	if err := common.Uninstall(manifest); err != nil {
		logger.Error("Failed to finalize platform resources", err)
	}
	return nil
}

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, ks *v1alpha1.KnativeServing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	ks.Status.InitializeConditions()
	ks.Status.ObservedGeneration = ks.Generation

	logger.Infow("Reconciling KnativeServing", "status", ks.Status)

	if err := common.IsVersionValidMigrationEligible(ks); err != nil {
		ks.Status.MarkVersionMigrationNotEligible(err.Error())
		return nil
	}
	ks.Status.MarkVersionMigrationEligible()

	if err := r.extension.Reconcile(ctx, ks); err != nil {
		return err
	}
	stages := common.Stages{
		common.AppendTarget,
		ingress.AppendTargetIngresses,
		r.filterDisabledIngresses,
		r.transform,
		common.Install,
		common.CheckDeployments,
		common.DeleteObsoleteResources(ctx, ks, r.installed),
	}
	platformManifests, err := r.extension.Manifests(ks)
	if err != nil {
		return err
	}
	manifest := r.manifest.Append(platformManifests...)
	return stages.Execute(ctx, &manifest, ks)
}

// filterDisabledIngresses removes the disabled ingresses from the manifests
func (r *Reconciler) filterDisabledIngresses(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	ks := instance.(*v1alpha1.KnativeServing)
	*manifest = manifest.Filter(ingress.Filters(ks))
	return nil
}

// transform mutates the passed manifest to one with common, component
// and platform transformations applied
func (r *Reconciler) transform(ctx context.Context, manifest *mf.Manifest, comp v1alpha1.KComponent) error {
	logger := logging.FromContext(ctx)
	instance := comp.(*v1alpha1.KnativeServing)
	extra := []mf.Transformer{
		ksc.CustomCertsTransform(instance, logger),
		ksc.AggregationRuleTransform(manifest.Client),
	}
	extra = append(extra, r.extension.Transformers(instance)...)
	extra = append(extra, ksc.IngressServiceTransform())
	extra = append(extra, ingress.Transformers(ctx, instance)...)
	return common.Transform(ctx, manifest, instance, extra...)
}

func (r *Reconciler) installed(ctx context.Context, instance v1alpha1.KComponent) (*mf.Manifest, error) {
	// Create new, empty manifest with valid client and logger
	installed := r.manifest.Append()
	stages := common.Stages{common.AppendInstalled, ingress.AppendInstalledIngresses, r.transform}
	err := stages.Execute(ctx, &installed, instance)
	return &installed, err
}
