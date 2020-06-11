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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"

	mf "github.com/manifestival/manifestival"
	"golang.org/x/mod/semver"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const (
	KoEnvKey = "KO_DATA_PATH"
)

var cache = map[string]mf.Manifest{}

// FetchManifest returns a possibly cached manifest
func FetchManifest(path string) (mf.Manifest, error) {
	if m, ok := cache[path]; ok {
		return m, nil
	}
	result, err := mf.NewManifest(path)
	if err == nil {
		cache[path] = result
	}
	return result, err
}

// ManifestPath returns the manifest path for Knative based a provided version and component
func ManifestPath(version, kcomponent string) string {
	koDataDir := os.Getenv(KoEnvKey)
	return filepath.Join(koDataDir, kcomponent, version)
}

// TargetRelease returns spec.Version, status.Version, or the
// operator's latest release
func TargetRelease(instance v1alpha1.KComponent) string {
	if instance.GetSpec().GetVersion() != "" {
		return instance.GetSpec().GetVersion()
	}
	if instance.GetStatus().GetVersion() != "" {
		return instance.GetStatus().GetVersion()
	}
	return LatestRelease(PathElement(instance))
}

func PathElement(instance v1alpha1.KComponent) string {
	switch instance.(type) {
	case *v1alpha1.KnativeServing:
		return "knative-serving"
	case *v1alpha1.KnativeEventing:
		return "knative-eventing"
	}
	return ""
}

// sanitizeSemver always adds `v` in front of the version.
// x.y.z is the standard format we use as the semantic version for Knative. The letter `v` is added for
// comparison purpose.
func sanitizeSemver(version string) string {
	return fmt.Sprintf("v%s", version)
}

// ListReleases returns the all the available release versions available under kodata directory for Knative component.
func allReleases(kComponent string) ([]string, error) {
	// List all the directories available under kodata
	koDataDir := os.Getenv(KoEnvKey)
	pathname := filepath.Join(koDataDir, kComponent)
	fileList, err := ioutil.ReadDir(pathname)
	if err != nil {
		return nil, err
	}

	releaseTags := make([]string, 0, len(fileList))
	for _, file := range fileList {
		name := path.Join(pathname, file.Name())
		pathDirOrFile, err := os.Stat(name)
		if err != nil {
			return nil, err
		}
		if pathDirOrFile.IsDir() {
			releaseTags = append(releaseTags, file.Name())
		}
	}
	if len(releaseTags) == 0 {
		return nil, fmt.Errorf("unable to find any version number for %s", kComponent)
	}

	// This function makes sure the versions are sorted in a descending order.
	sort.Slice(releaseTags, func(i, j int) bool {
		// The index i is the one after the index j. If i is more recent than j, return true to swap.
		return semver.Compare(sanitizeSemver(releaseTags[i]), sanitizeSemver(releaseTags[j])) == 1
	})

	return releaseTags, nil
}

// LatestRelease returns the latest release tag available under kodata directory for Knative component.
func LatestRelease(kcomponent string) string {
	vers, err := allReleases(kcomponent)
	if err != nil {
		panic(err)
	}
	// The versions are in a descending order, so the first one will be the latest version.
	return vers[0]
}
