/*
Copyright 2022 The Knative Authors

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

package manifests

import (
	"context"

	mf "github.com/manifestival/manifestival"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
	"knative.dev/operator/pkg/reconciler/knativeeventing/source"
	ksc "knative.dev/operator/pkg/reconciler/knativeserving/common"
	"knative.dev/operator/pkg/reconciler/knativeserving/ingress"
)

func Install(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	err := common.Install(ctx, manifest, instance)
	if err != nil {
		return err
	}
	return nil
}

func SetManifestPaths(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	status := instance.GetStatus()
	path := common.TargetManifestPathArray(instance)
	version := common.TargetVersion(instance)
	addedPath := ingress.GetIngressPath(version, ksc.ConvertToKS(instance))
	switch instance.(type) {
	case *v1beta1.KnativeEventing:
		addedPath = source.GetSourcePath(version, source.ConvertToKE(instance))
	}
	if addedPath != "" {
		path = append(path, addedPath)
	}
	status.SetManifests(path)
	return nil
}
