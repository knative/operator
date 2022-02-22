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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
)

const providerLabel = "networking.knative.dev/ingress-provider"

func ingressFilter(name string) mf.Predicate {
	return func(u *unstructured.Unstructured) bool {
		provider, hasLabel := u.GetLabels()[providerLabel]
		if !hasLabel {
			return true
		}
		return provider == name
	}
}

// noneFilter drops all ingresses but allows everything else.
func noneFilter(u *unstructured.Unstructured) bool {
	_, hasLabel := u.GetLabels()[providerLabel]
	return !hasLabel
}

// Filters makes sure the disabled ingress resources are removed from the manifest.
func Filters(ks *v1beta1.KnativeServing) mf.Predicate {
	var filters []mf.Predicate
	if ks.Spec.Ingress == nil {
		return istioFilter
	}
	if ks.Spec.Ingress.Istio.Enabled {
		filters = append(filters, istioFilter)
	}
	if ks.Spec.Ingress.Kourier.Enabled {
		filters = append(filters, kourierFilter)
	}
	if ks.Spec.Ingress.Contour.Enabled {
		filters = append(filters, contourFilter)
	}
	if len(filters) == 0 {
		return noneFilter
	}
	return mf.Any(filters...)
}

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

func getIngress(version string) (mf.Manifest, error) {
	// If we can not determine the version, append no ingress manifest.
	if version == "" {
		return mf.Manifest{}, nil
	}
	koDataDir := os.Getenv(common.KoEnvKey)
	// Ingresses are saved in the directory named major.minor. We remove the patch number.
	ingressVersion := common.LATEST_VERSION
	if !strings.EqualFold(version, common.LATEST_VERSION) {
		ingressVersion = semver.MajorMinor(common.SanitizeSemver(version))[1:]
	}

	// This line can make sure a valid available ingress version is returned.
	ingressVersion = common.GetLatestIngressRelease(ingressVersion)
	ingressPath := filepath.Join(koDataDir, "ingress", ingressVersion)
	return common.FetchManifest(ingressPath)
}

// AppendTargetIngresses appends the manifests of ingresses to be installed
func AppendTargetIngresses(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	m, err := getIngress(common.TargetVersion(instance))
	if err == nil {
		*manifest = manifest.Append(m)
	}

	if len(instance.GetSpec().GetManifests()) != 0 {
		// If spec.manifests is not empty, it is possible that the ingress is not available with the specified version.
		// The user can specify the ingress link in the spec.manifests.
		return nil
	}
	return err
}

// AppendInstalledIngresses appends the installed manifests of ingresses
func AppendInstalledIngresses(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	version := instance.GetStatus().GetVersion()
	if version == "" {
		version = common.TargetVersion(instance)
	}

	m, err := getIngress(version)
	if err == nil {
		*manifest = manifest.Append(m)
	}

	// It is possible that the ingress is not available with the specified version.
	// If the user specified a version with a minor version, which is not supported by the current operator, as long as
	// spec.manifests contains all the manifest links, the operator can still work. This function can always return nil,
	// even if the ingress is not available.
	return nil
}

func hasProviderLabel(u *unstructured.Unstructured) bool {
	if _, hasLabel := u.GetLabels()[providerLabel]; hasLabel {
		return true
	}
	return false
}
