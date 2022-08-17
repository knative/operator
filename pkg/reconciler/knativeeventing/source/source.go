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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	mf "github.com/manifestival/manifestival"
	"golang.org/x/mod/semver"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
)

func getSource(manifest *mf.Manifest, path string) (mf.Manifest, error) {
	if path == "" {
		return mf.Manifest{}, nil
	}
	return common.FetchManifest(path)
}

func getAllSourcePath(version string) string {
	koDataDir := os.Getenv(common.KoEnvKey)
	sourceVersion := common.LATEST_VERSION
	if !strings.EqualFold(version, common.LATEST_VERSION) {
		sourceVersion = semver.MajorMinor(common.SanitizeSemver(version))[1:]
	}

	sourcePath := filepath.Join(koDataDir, "eventing-source", sourceVersion)
	// List all the directories under the sourcePath, because we will append all the paths for eventing sources
	fileList, err := ioutil.ReadDir(sourcePath)
	if err != nil {
		return ""
	}

	var urls []string
	for _, file := range fileList {
		name := path.Join(sourcePath, file.Name())
		pathDirOrFile, err := os.Stat(name)
		if err != nil {
			continue
		}
		if pathDirOrFile.IsDir() {
			urls = append(urls, name)
		}
	}
	return strings.Join(urls, common.COMMA)
}

func getSourcePath(version string, ke *v1beta1.KnativeEventing) string {
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

	if ke.Spec.Source.Ceph.Enabled {
		url := filepath.Join(sourcePath, "ceph")
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
func AppendTargetSources(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	version := common.TargetVersion(instance)
	sourcePath := getSourcePath(version, convertToKE(instance))
	m, err := getSource(manifest, sourcePath)
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

// AppendAllSources appends all the manifests of the eventing sources
func AppendAllSources(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	version := instance.GetStatus().GetVersion()
	if version == "" {
		version = common.TargetVersion(instance)
	}
	sourcePath := getAllSourcePath(version)
	m, err := getSource(manifest, sourcePath)
	if err == nil {
		*manifest = manifest.Append(m)
	}

	// It is possible that the eventing source is not available with the specified version.
	// If the user specified a version with a minor version, which is not supported by the current operator, the operator
	// can still work, as long as spec.manifests contains all the manifest links. This function can always return nil,
	// even if the eventing source is not available.
	return nil
}

func convertToKE(instance base.KComponent) *v1beta1.KnativeEventing {
	ke := &v1beta1.KnativeEventing{}
	switch instance := instance.(type) {
	case *v1beta1.KnativeEventing:
		ke = instance
	}
	return ke
}
