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
	"k8s.io/client-go/kubernetes"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	clientset "knative.dev/operator/pkg/client/clientset/versioned"
	knsreconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeserving"
	"knative.dev/operator/pkg/reconciler/common"
	ksc "knative.dev/operator/pkg/reconciler/knativeserving/common"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
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

	if err := r.extension.Finalize(ctx, original); err != nil {
		logger.Error("Failed to finalize platform resources", err)
	}
	logger.Info("Deleting cluster-scoped resources")
	manifest, err := r.installed(ctx, original)
	if err != nil {
		logger.Error("Unable to fetch installed manifest; no cluster-scoped resources will be finalized", err)
		return nil
	}
	return common.Uninstall(manifest)
}

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, ks *v1alpha1.KnativeServing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	ks.Status.InitializeConditions()
	ks.Status.ObservedGeneration = ks.Generation

	// TODO: I'm sure there's a more elegant way to do this
	ctx = context.WithValue(ctx, kubeclient.Key{}, r.kubeClientSet)

	logger.Infow("Reconciling KnativeServing", "status", ks.Status)
	if err := r.extension.Reconcile(ctx, ks); err != nil {
		return err
	}
	stages := common.Stages{
		common.AppendTarget,
		r.transform,
		common.Install,
		common.CheckDeployments,
		r.deleteObsoleteResources(ctx, ks),
	}
	manifest := r.manifest.Append()
	return stages.Execute(ctx, &manifest, ks)
}

// transform mutates the passed manifest to one with common, component
// and platform transformations applied
func (r *Reconciler) transform(ctx context.Context, manifest *mf.Manifest, comp v1alpha1.KComponent) error {
	logger := logging.FromContext(ctx)
	instance := comp.(*v1alpha1.KnativeServing)
	extra := []mf.Transformer{
		ksc.GatewayTransform(instance, logger),
		ksc.CustomCertsTransform(instance, logger),
		ksc.HighAvailabilityTransform(instance, logger),
		ksc.AggregationRuleTransform(manifest.Client),
	}
	extra = append(extra, r.extension.Transformers(instance)...)
	return common.Transform(ctx, manifest, instance, extra...)
}

// deleteObsoleteResources returns a Stage after calculating the
// installed manifest from the instance, but *before* any other stages
// might mutate the instance's status.version.
func (r *Reconciler) deleteObsoleteResources(ctx context.Context, instance v1alpha1.KComponent) common.Stage {
	if common.TargetVersion(instance) == instance.GetStatus().GetVersion() {
		return common.NoOp
	}
	logger := logging.FromContext(ctx)
	installed, err := r.installed(ctx, instance)
	if err != nil {
		logger.Error("Unable to obtain the installed manifest; obsolete resources may linger", err)
		return common.NoOp
	}
	return func(_ context.Context, manifest *mf.Manifest, _ v1alpha1.KComponent) error {
		return installed.Filter(mf.None(mf.In(*manifest))).Delete()
	}
}

func (r *Reconciler) installed(ctx context.Context, instance v1alpha1.KComponent) (*mf.Manifest, error) {
	// Create new, empty manifest with valid client and logger
	installed := r.manifest.Append()
	// TODO: add ingress, etc
	stages := common.Stages{common.AppendInstalled, r.transform}
	err := stages.Execute(ctx, &installed, instance)
	return &installed, err
}
