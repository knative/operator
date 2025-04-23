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

package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mf "github.com/manifestival/manifestival"
	"golang.org/x/mod/semver"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
	servingcommon "knative.dev/operator/pkg/reconciler/knativeserving/common"
)

// AppendTargetSecurity appends the manifests of the security guard to be installed
func AppendTargetSecurity(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	version := common.TargetVersion(instance)
	m, err := getSecurity(version, servingcommon.ConvertToKS(instance))

	if err == nil {
		*manifest = manifest.Append(m)
	}

	if len(instance.GetSpec().GetManifests()) != 0 {
		// If spec.manifests is not empty, it is possible that the securityguard is not available with the specified version.
		// The user can specify the securityguard link in the spec.manifests.
		return nil
	}
	return err
}

// Transformers returns a list of transformers based on the enabled security options
func Transformers(ctx context.Context, ks *v1beta1.KnativeServing) []mf.Transformer {
	var transformers []mf.Transformer
	if ks.Spec.Security == nil {
		return transformers
	}

	if ks.Spec.Security.SecurityGuard.Enabled {
		transformers = append(transformers, securityGuardTransformers(ctx, ks)...)
	}

	return transformers
}

func getSecurity(version string, ks *v1beta1.KnativeServing) (mf.Manifest, error) {
	if ks.Spec.Security == nil || !ks.Spec.Security.SecurityGuard.Enabled {
		// If no security option is defined, return an empty string.
		return mf.Manifest{}, nil
	}

	// If we can not determine the version, append no security guard manifest.
	if version == "" {
		return mf.Manifest{}, nil
	}
	koDataDir := os.Getenv(common.KoEnvKey)

	// Security Guard is saved in the directory named major.minor. We remove the patch number.
	var servingVersion string
	if !strings.EqualFold(version, common.LATEST_VERSION) {
		servingVersion = semver.MajorMinor(common.SanitizeSemver(version))[1:]
	} else {
		// This line can make sure a valid available securityguard version is returned.
		servingVersion = common.GetLatestRelease(&v1beta1.KnativeServing{}, "")
		servingVersion = semver.MajorMinor(common.SanitizeSemver(servingVersion))[1:]
	}

	// Find the specific security guard version via the hash map
	sgVersion, ok := SecurityGuardVersion[common.SanitizeSemver(servingVersion)]
	if !ok {
		return mf.Manifest{}, fmt.Errorf("the current version of Knative Serving is %v. You need to install the "+
			"version 1.8 or above to support the security guard", servingVersion)
	}

	sgPath := filepath.Join(koDataDir, "security-guard", sgVersion)
	return common.FetchManifest(sgPath)
}
