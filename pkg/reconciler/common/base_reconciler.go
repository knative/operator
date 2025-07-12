/*
Copyright 2020 The Knative Authors

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

package common

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"

	"knative.dev/operator/pkg/apis/operator/base"
)

// BaseReconciler provides common functionality for component reconcilers
type BaseReconciler struct {
	manifest  mf.Manifest
	extension Extension
}

// NewBaseReconciler creates a new base reconciler
func NewBaseReconciler(manifest mf.Manifest, extension Extension) *BaseReconciler {
	return &BaseReconciler{
		manifest:  manifest,
		extension: extension,
	}
}

// ReconcileComponent provides the common reconciliation logic for components
func (r *BaseReconciler) ReconcileComponent(ctx context.Context, instance base.KComponent, stages Stages) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	// Note: InitializeConditions and SetObservedGeneration are specific to each component type
	// and should be called by the concrete reconciler implementations

	logger.Infow("Reconciling component", "status", instance.GetStatus())

	if err := IsVersionValidMigrationEligible(instance); err != nil {
		instance.GetStatus().MarkVersionMigrationNotEligible(err.Error())
		return nil
	}
	instance.GetStatus().MarkVersionMigrationEligible()

	if err := r.extension.Reconcile(ctx, instance); err != nil {
		return err
	}

	manifest := r.manifest.Append()
	return stages.Execute(ctx, &manifest, instance)
}

// GetManifest returns the base manifest
func (r *BaseReconciler) GetManifest() mf.Manifest {
	return r.manifest
}

// GetExtension returns the extension
func (r *BaseReconciler) GetExtension() Extension {
	return r.extension
}
