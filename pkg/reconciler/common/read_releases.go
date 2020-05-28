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
)

// ListReleases returns the all the available release verions available under kodata directory for Knative component.
func ListReleases(kComponent string) ([]string, error) {
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
	if len(releaseTags) == 0 {
		return releaseTags, fmt.Errorf("unable to find available version number for the Knative Serving")
	}

	sort.Slice(releaseTags, func(i, j int) bool {
		return releaseTags[i] > releaseTags[j]
	})
	return releaseTags, nil
}

// GetLatestRelease returns the latest release tag available under kodata directory for Knative component.
func GetLatestRelease(kComponent string) (string, error) {
	releaseTag := ""
	releaseTags, err := ListReleases(kComponent)
	if err != nil {
		return releaseTag, err
	}

	releaseTag = releaseTags[0]
	return releaseTag, nil
}

func RetrieveManifest(ctx context.Context, version string, mfClient mf.Client) (mf.Manifest, error) {
	logger := logging.FromContext(ctx)
	koDataDir := os.Getenv("KO_DATA_PATH")
	manifestDir := fmt.Sprintf("knative-serving/%s", version)
	manifest, err := mf.NewManifest(filepath.Join(koDataDir, manifestDir),
		mf.UseClient(mfClient),
		mf.UseLogger(zapr.NewLogger(logger.Desugar()).WithName("manifestival")))

	if err != nil {
		return manifest, err
	}

	if len(manifest.Resources()) == 0 {
		return manifest, fmt.Errorf("unable to find the manifest for the Knative Serving version %s", version)
	}

	return manifest, nil
}
