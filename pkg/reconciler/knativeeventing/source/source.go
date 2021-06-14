/*
Copyright 2021 The Knative Authors

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

package source

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	mf "github.com/manifestival/manifestival"
	"golang.org/x/mod/semver"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/operator/pkg/reconciler/common"
)

func getSource(manifest *mf.Manifest, path string) error {
	if path == "" {
		return nil
	}
	m, err := common.FetchManifest(path)
	if err != nil {
		return err
	}
	*manifest = manifest.Append(m)
	return nil
}

func getSourcePath(version string, ke *v1alpha1.KnativeEventing) string {
	if ke.Spec.Source == nil {
		// If no eventing source is defined, return an empty string.
		return ""
	}

	koDataDir := os.Getenv(common.KoEnvKey)
	sourceVersion := common.LATEST_VERSION
	if !strings.EqualFold(version, common.LATEST_VERSION) {
		sourceVersion = semver.MajorMinor(common.SanitizeSemver(version))[1:]
	}

	// This line can make sure a valid available source version is returned.
	sourcePath := filepath.Join(koDataDir, "eventing-source", sourceVersion)
	var urls []string

	if ke.Spec.Source.Awssqs.Enabled {
		url := filepath.Join(sourcePath, "awssqs")
		urls = append(urls, url)
	}
	if ke.Spec.Source.Ceph.Enabled {
		url := filepath.Join(sourcePath, "ceph")
		urls = append(urls, url)
	}
	if ke.Spec.Source.Couchdb.Enabled {
		url := filepath.Join(sourcePath, "couchdb")
		urls = append(urls, url)
	}
	if ke.Spec.Source.Github.Enabled {
		url := filepath.Join(sourcePath, "github")
		urls = append(urls, url)
	}
	if ke.Spec.Source.Gitlab.Enabled {
		url := filepath.Join(sourcePath, "gitlab")
		urls = append(urls, url)
	}
	if ke.Spec.Source.Kafka.Enabled {
		url := filepath.Join(sourcePath, "kafka")
		urls = append(urls, url)
	}
	if ke.Spec.Source.Natss.Enabled {
		url := filepath.Join(sourcePath, "natss")
		urls = append(urls, url)
	}
	if ke.Spec.Source.Prometheus.Enabled {
		url := filepath.Join(sourcePath, "prometheus")
		urls = append(urls, url)
	}
	if ke.Spec.Source.Rabbitmq.Enabled {
		url := filepath.Join(sourcePath, "rabbitmq")
		urls = append(urls, url)
	}
	if ke.Spec.Source.Redis.Enabled {
		url := filepath.Join(sourcePath, "redis")
		urls = append(urls, url)
	}
	return strings.Join(urls, common.COMMA)
}

// AppendTargetSources appends the manifests of the eventing sources to be installed
func AppendTargetSources(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	version := common.TargetVersion(instance)
	sourcePath := getSourcePath(version, convertToKE(instance))
	return getSource(manifest, sourcePath)
}

// AppendInstalledSources appends the installed manifests of the eventing sources
func AppendInstalledSources(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	version := instance.GetStatus().GetVersion()
	if version == "" {
		version = common.TargetVersion(instance)
	}
	sourcePath := getSourcePath(version, convertToKE(instance))
	return getSource(manifest, sourcePath)
}

func convertToKE(instance v1alpha1.KComponent) *v1alpha1.KnativeEventing {
	ke := &v1alpha1.KnativeEventing{}
	switch instance := instance.(type) {
	case *v1alpha1.KnativeEventing:
		ke = instance
	}
	return ke
}
