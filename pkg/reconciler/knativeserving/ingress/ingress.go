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

package ingress

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	mf "github.com/manifestival/manifestival"
	"golang.org/x/mod/semver"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
)

const providerLabel = "networking.knative.dev/ingress-provider"

// Transformers returns a list of transformers based on the enabled ingresses
func Transformers(ctx context.Context, ks *v1beta1.KnativeServing) []mf.Transformer {
	if ks.Spec.Ingress == nil {
		return istioTransformers(ctx, ks)
	}
	var transformers []mf.Transformer
	if ks.Spec.Ingress.Istio.Enabled {
		transformers = append(transformers, istioTransformers(ctx, ks)...)
	}
	if ks.Spec.Ingress.Kourier.Enabled {
		transformers = append(transformers, kourierTransformers(ctx, ks)...)
	}
	if ks.Spec.Ingress.Contour.Enabled {
		transformers = append(transformers, contourTransformers(ctx, ks)...)
	}
	return transformers
}

func getIngress(path string) (mf.Manifest, error) {
	if path == "" {
		return mf.Manifest{}, nil
	}
	return common.FetchManifest(path)
}

func getIngressPath(version string, ks *v1beta1.KnativeServing) string {
	var urls []string
	koDataDir := os.Getenv(common.KoEnvKey)
	sourceVersion := common.LATEST_VERSION
	if !strings.EqualFold(version, common.LATEST_VERSION) {
		sourceVersion = semver.MajorMinor(common.SanitizeSemver(version))[1:]
	}

	// This line can make sure a valid available source version is returned.
	ingressPath := filepath.Join(koDataDir, "ingress", sourceVersion)
	if ks.Spec.Ingress == nil {
		url := filepath.Join(ingressPath, "istio")
		urls = append(urls, url)
		return strings.Join(urls, common.COMMA)
	}

	if ks.Spec.Ingress.Istio.Enabled {
		url := filepath.Join(ingressPath, "istio")
		urls = append(urls, url)
	}
	if ks.Spec.Ingress.Contour.Enabled {
		url := filepath.Join(ingressPath, "contour")
		urls = append(urls, url)
	}
	if ks.Spec.Ingress.Kourier.Enabled {
		url := filepath.Join(ingressPath, "kourier")
		urls = append(urls, url)
	}

	if len(urls) == 0 {
		url := filepath.Join(ingressPath, "istio")
		urls = append(urls, url)
	}

	return strings.Join(urls, common.COMMA)
}

// AppendTargetIngress appends the manifests of the ingress to be installed
func AppendTargetIngress(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	version := common.TargetVersion(instance)
	ingressPath := getIngressPath(version, convertToKS(instance))
	m, err := getIngress(ingressPath)
	if err == nil {
		*manifest = manifest.Append(m)
	}
	if len(instance.GetSpec().GetManifests()) != 0 {
		// If spec.manifests is not empty, it is possible that the eventing source is not available with the
		// specified version. The user can specify the eventing source link in the spec.manifests.
		return nil
	}
	return err
}

// AppendInstalledIngresses appends all the manifests of the ingresses
func AppendInstalledIngresses(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	version := instance.GetStatus().GetVersion()
	if version == "" {
		version = common.TargetVersion(instance)
	}
	ingressPath := getIngressPath(version, convertToKS(instance))
	m, err := getIngress(ingressPath)
	if err == nil {
		*manifest = manifest.Append(m)
	}

	// It is possible that the ingress is not available with the specified version.
	// If the user specified a version with a minor version, which is not supported by the current operator, the operator
	// can still work, as long as spec.manifests contains all the manifest links. This function can always return nil,
	// even if the ingress is not available.
	return nil
}

func convertToKS(instance base.KComponent) *v1beta1.KnativeServing {
	ks := &v1beta1.KnativeServing{}
	switch instance := instance.(type) {
	case *v1beta1.KnativeServing:
		ks = instance
	}
	return ks
}
