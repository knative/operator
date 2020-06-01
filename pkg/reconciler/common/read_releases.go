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

	"golang.org/x/mod/semver"
)

const (
	KoEnvKey = "KO_DATA_PATH"
)

// RetrieveManifestPath returns the manifest path for Knative based a provided version and component
func RetrieveManifestPath(version, kcomponent string) string {
	koDataDir := os.Getenv(KoEnvKey)
	return filepath.Join(koDataDir, kcomponent, version)
}

func isFirstVersionMoreRecent(former, latter string) bool {
	return semver.Compare(addMissingLetterV(former), addMissingLetterV(latter)) == 1
}

func addMissingLetterV(version string) string {
	if version == "" || version[0] == 'v' {
		return version
	}
	return fmt.Sprintf("v%s", version)
}

// ListReleases returns the all the available release versions available under kodata directory for Knative component.
func ListReleases(kComponent string) []string {
	releaseTags := []string{}
	// List all the directories available under kodata
	koDataDir := os.Getenv(KoEnvKey)
	pathname := filepath.Join(koDataDir, kComponent)
	fileList, err := ioutil.ReadDir(pathname)
	if err != nil {
		panic(err)
	}
	for _, file := range fileList {
		name := path.Join(pathname, file.Name())
		pathDirOrFile, err := os.Stat(name)
		if err != nil {
			panic(err)
		}
		if pathDirOrFile.IsDir() {
			releaseTags = append(releaseTags, file.Name())
		}
	}
	if len(releaseTags) == 0 {
		panic(fmt.Errorf("unable to find any version number for %s", kComponent))
	}

	// This function makes sure the versions are sorted in a descending order.
	sort.Slice(releaseTags, func(i, j int) bool {
		// The index i is the one after the index j. If i is more recent than j, return true to swap.
		return isFirstVersionMoreRecent(releaseTags[i], releaseTags[j])
	})

	return releaseTags
}

// GetLatestRelease returns the latest release tag available under kodata directory for Knative component.
func GetLatestRelease(kcomponent string) string {
	// The versions are in a descending order, so the first one will be the latest version.
	return ListReleases(kcomponent)[0]
}
