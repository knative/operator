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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
)

func listKodataReleases(kComponent string) ([]string, error) {
	releaseTags := []string{}
	// List all the directories available under kodata
	koDataDir := os.Getenv("KO_DATA_PATH")
	pathname := filepath.Join(koDataDir, kComponent)
	fileList, err := ioutil.ReadDir(pathname)
	if err != nil {
		return releaseTags, err
	}
	for _, file := range fileList {
		name := path.Join(pathname, file.Name())
		pathDirOrFile, err := os.Stat(name)
		if err != nil {
			return releaseTags, err
		}
		if pathDirOrFile.IsDir() {
			releaseTags = append(releaseTags, file.Name())
		}
	}
	sort.Slice(releaseTags, func(i, j int) bool {
		return releaseTags[i] > releaseTags[j]
	})
	return releaseTags, nil
}

// GetLatestKodataReleaseTag returns the latest release tag available under kodata directory for Knative component.
func GetLatestKodataReleaseTag(kComponent string) (string, error) {
	releaseTag := ""
	releaseTags, err := listKodataReleases(kComponent)
	if err != nil {
		return releaseTag, err
	}
	if len(releaseTags) == 0 {
		return releaseTag, nil
	}

	releaseTag = releaseTags[0]
	return releaseTag, nil
}
