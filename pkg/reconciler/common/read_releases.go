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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/go-logr/zapr"
	mf "github.com/manifestival/manifestival"
	"knative.dev/pkg/logging"

	"golang.org/x/mod/semver"
)

const (
	KoEnvKey = "KO_DATA_PATH"
)

// retrieveManifestPath returns the manifest path for Knative based a provided version and component
func retrieveManifestPath(version, kcomponent string) string {
	koDataDir := os.Getenv(KoEnvKey)
	return filepath.Join(koDataDir, kcomponent, version)
}

// sanitizeSemver always adds `v` in front of the version.
// x.y.z is the standard format we use as the semantic version for Knative. The letter `v` is added for
// comparison purpose.
func sanitizeSemver(version string) string {
	return fmt.Sprintf("v%s", version)
}

// ListReleases returns the all the available release versions available under kodata directory for Knative component.
func ListReleases(kComponent string) ([]string, error) {
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

// GetLatestRelease returns the latest release tag available under kodata directory for Knative component.
func GetLatestRelease(kcomponent string) string {
	vers, err := ListReleases(kcomponent)
	if err != nil {
		panic(err)
	}
	// The versions are in a descending order, so the first one will be the latest version.
	return vers[0]
}

// RetrieveManifest returns the manifest for Knative component based a provided version
func RetrieveManifest(ctx context.Context, version, component string, mfClient mf.Client) (mf.Manifest, error) {
	logger := logging.FromContext(ctx)
	manifest, err := mf.NewManifest(retrieveManifestPath(version, component),
		mf.UseClient(mfClient),
		mf.UseLogger(zapr.NewLogger(logger.Desugar()).WithName("manifestival")))

	if err != nil {
		return manifest, err
	}

	if len(manifest.Resources()) == 0 {
		return manifest, fmt.Errorf("unable to find the manifest for %s at the version %s", component, version)
	}

	return manifest, nil
}
