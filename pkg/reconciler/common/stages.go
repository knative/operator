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
	"strings"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/pkg/logging"
)

// Stage represents a step in the reconcile process
type Stage func(context.Context, *mf.Manifest, base.KComponent) error

// Stages are a list of steps
type Stages []Stage

// Execute each stage in sequence until one returns an error
func (stages Stages) Execute(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	for _, stage := range stages {
		if err := stage(ctx, manifest, instance); err != nil {
			return err
		}
	}
	return nil
}

// NoOp does nothing
func NoOp(context.Context, *mf.Manifest, base.KComponent) error {
	return nil
}

// AppendTarget mutates the passed manifest by appending one
// appropriate for the passed KComponent
func AppendTarget(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	m, err := TargetManifest(instance)
	if err != nil {
		instance.GetStatus().MarkInstallFailed(err.Error())
		return err
	}
	*manifest = manifest.Append(m)
	return nil
}

// AppendAdditionalManifests mutates the passed manifest by appending the manifests specified with the
// field spec.additionalManifests.
func AppendAdditionalManifests(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	m, err := TargetAdditionalManifest(instance)
	if err != nil {
		instance.GetStatus().MarkInstallFailed(err.Error())
		return err
	}
	// If we get the same resource in the additional manifests, we will remove the one in the existing manifest.
	if len(m.Resources()) != 0 {
		*manifest = manifest.Filter(mf.Not(mf.In(m))).Append(m)
	}
	return nil
}

// AppendInstalled mutates the passed manifest by appending one
// appropriate for the passed KComponent, which may not be the one
// corresponding to status.version
func AppendInstalled(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	logger := logging.FromContext(ctx)
	m, err := InstalledManifest(instance)
	if err != nil {
		// TODO: return the oldest instead of the latest?
		logger.Error("Unable to fetch installed manifest, trying target", err)
		m, err = TargetManifest(instance)
	}
	if err != nil {
		return err
	}
	*manifest = manifest.Append(m)
	return nil
}

// ManifestFetcher returns a manifest appropriate for the instance
type ManifestFetcher func(ctx context.Context, instance base.KComponent) (*mf.Manifest, error)

// DeleteObsoleteResources returns a Stage after calculating the
// installed manifest from the instance. This is meant to be called
// *before* executing the reconciliation stages so that the proper
// manifest is captured in a closure before any stage might mutate the
// instance status, e.g. Install.
func DeleteObsoleteResources(ctx context.Context, instance base.KComponent, fetch ManifestFetcher) Stage {
	version := TargetVersion(instance)
	if version == instance.GetStatus().GetVersion() && len(instance.GetSpec().GetAdditionalManifests()) == 0 &&
		len(instance.GetSpec().GetManifests()) == 0 &&
		targetManifestPath(instance) == strings.Join(installedManifestPath(version, instance), COMMA) {
		return NoOp
	}
	logger := logging.FromContext(ctx)
	installed, err := fetch(ctx, instance)
	if err != nil {
		logger.Error("Unable to obtain the installed manifest; obsolete resources may linger", err)
		return NoOp
	}
	return func(_ context.Context, manifest *mf.Manifest, _ base.KComponent) error {
		return installed.Filter(mf.NoCRDs, mf.Not(mf.In(*manifest))).Delete()
	}
}
