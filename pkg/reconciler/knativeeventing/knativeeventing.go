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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	clientset "knative.dev/operator/pkg/client/clientset/versioned"
	knereconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1beta1/knativeeventing"
	"knative.dev/operator/pkg/reconciler/common"
	kec "knative.dev/operator/pkg/reconciler/knativeeventing/common"
	"knative.dev/operator/pkg/reconciler/knativeeventing/source"
	"knative.dev/operator/pkg/reconciler/manifests"
)

// Reconciler implements controller.Reconciler for KnativeEventing resources.
type Reconciler struct {
	// kubeClientSet allows us to talk to the k8s for core APIs
	kubeClientSet kubernetes.Interface
	// kubeClientSet allows us to talk to the k8s for operator APIs
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
var _ knereconciler.Interface = (*Reconciler)(nil)
var _ knereconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind removes all resources after deletion of a KnativeEventing.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *v1beta1.KnativeEventing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	// Clean up the cache, if the Serving CR is deleted.
	common.ClearCache()

	// List all KnativeEventings to determine if cluster-scoped resources should be deleted.
	kes, err := r.operatorClientSet.OperatorV1beta1().KnativeEventings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all KnativeEventings: %w", err)
	}

	for _, ke := range kes.Items {
		if ke.GetDeletionTimestamp().IsZero() {
			// Not deleting all KnativeEventings. Nothing to do here.
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

	if manifest == nil {
		return nil
	}

	// For optional resources like cert-manager's Certificates and Issuers, we don't want to fail
	// finalization when such operator is not installed, so we split the resources in
	// - optional resources (TLS resources, etc)
	// - resources (core k8s resources)
	//
	// Then, we delete `resources` first and after we delete optional resources while also ignoring
	// errors returned when such operators are not installed.

	optionalResourcesPred := mf.Any(tlsResourcesPred)

	optionalResources := manifest.Filter(optionalResourcesPred)
	resources := manifest.Filter(mf.Not(optionalResourcesPred))

	if err = common.Uninstall(&resources); err != nil {
		logger.Error("Failed to finalize platform resources", err)
	}

	if err := common.Uninstall(&optionalResources); err != nil && !meta.IsNoMatchError(err) {
		logger.Error("Failed to finalize platform resources", err)
	}

	return nil
}

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, ke *v1beta1.KnativeEventing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	ke.Status.InitializeConditions()
	ke.Status.ObservedGeneration = ke.Generation

	logger.Infow("Reconciling KnativeEventing", "status", ke.Status)

	if err := common.IsVersionValidMigrationEligible(ke); err != nil {
		ke.Status.MarkVersionMigrationNotEligible(err.Error())
		return nil
	}
	ke.Status.MarkVersionMigrationEligible()

	if err := r.extension.Reconcile(ctx, ke); err != nil {
		return err
	}
	stages := common.Stages{
		common.AppendTarget,
		source.AppendTargetSources,
		common.AppendAdditionalManifests,
		r.appendExtensionManifests,
		r.transform,
		r.handleTLSResources,
		manifests.Install,
		common.CheckDeployments,
		common.DeleteObsoleteResources(ctx, ke, r.installed),
	}
	manifest := r.manifest.Append()
	return stages.Execute(ctx, &manifest, ke)
}

// transform mutates the passed manifest to one with common, component
// and platform transformations applied
func (r *Reconciler) transform(ctx context.Context, manifest *mf.Manifest, comp base.KComponent) error {
	logger := logging.FromContext(ctx)
	instance := comp.(*v1beta1.KnativeEventing)
	extra := []mf.Transformer{
		kec.DefaultBrokerConfigMapTransform(instance, logger),
		kec.SinkBindingSelectionModeTransform(instance, logger),
		kec.ReplicasEnvVarsTransform(manifest.Client),
	}
	extra = append(extra, r.extension.Transformers(instance)...)
	return common.Transform(ctx, manifest, instance, extra...)
}

// injectNamespace mutates the namespace of all installed resources
func (r *Reconciler) injectNamespace(ctx context.Context, manifest *mf.Manifest, comp base.KComponent) error {
	return common.InjectNamespace(manifest, comp)
}

func (r *Reconciler) installed(ctx context.Context, instance base.KComponent) (*mf.Manifest, error) {
	paths := instance.GetStatus().GetManifests()
	installed, err := common.FetchManifestFromArray(paths)

	if err != nil {
		return &installed, err
	}
	installed = r.manifest.Append(installed)

	// Per the manifests, that have been installed in the cluster, we only need to inject the correct namespace
	// in the stages.
	stages := common.Stages{r.injectNamespace}
	err = stages.Execute(ctx, &installed, instance)
	return &installed, err
}

func (r *Reconciler) appendExtensionManifests(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	platformManifests, err := r.extension.Manifests(instance)
	if err != nil {
		return err
	}
	*manifest = manifest.Append(platformManifests...)
	return nil
}
