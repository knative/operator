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
	"strings"

	"github.com/go-logr/zapr"
	mf "github.com/manifestival/manifestival"

	"knative.dev/pkg/logging"
)

const (
	RELEASE_LINK = "https://github.com/knative/%s/releases/download/v%s/%s"
)

func listReleases(kComponent string) ([]string, error) {
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

// GetEarliestSupportedRelease returns the earliest supported release available under kodata directory for Knative component.
func GetEarliestSupportedRelease(kComponent string) (string, error) {
	releaseTag := ""
	releaseTags, err := listReleases(kComponent)
	if err != nil {
		return releaseTag, err
	}

	releaseTag = releaseTags[len(releaseTags) - 1]
	return releaseTag, nil
}

// GetLatestRelease returns the latest release tag available under kodata directory for Knative component.
func GetLatestRelease(kComponent string) (string, error) {
	releaseTag := ""
	releaseTags, err := listReleases(kComponent)
	if err != nil {
		return releaseTag, err
	}

	releaseTag = releaseTags[0]
	return releaseTag, nil
}

// RetrieveManifest returns the manifest for Knative based a provided version
func RetrieveManifest(ctx context.Context, version, component string, mfClient mf.Client,
	yamlList []string) (mf.Manifest, error) {
	logger := logging.FromContext(ctx)
	koDataDir := os.Getenv("KO_DATA_PATH")
	manifesrDir := fmt.Sprintf("knative-%s/%s", component, version)
	manifest, err := mf.NewManifest(filepath.Join(koDataDir, manifesrDir),
		mf.UseClient(mfClient),
		mf.UseLogger(zapr.NewLogger(logger.Desugar()).WithName("manifestival")))

	if err != nil {
		// If it is not available locally, we look up the yamls online
		for index, yaml := range yamlList {
			file := yaml
			if strings.Contains(yaml, "%s") {
				file = fmt.Sprintf(yaml, version)
			}
			fileLink := fmt.Sprintf(RELEASE_LINK, component, version, file)
			manifestYaml, err := mf.NewManifest(fileLink,
				mf.UseClient(mfClient),
				mf.UseLogger(zapr.NewLogger(logger.Desugar()).WithName("manifestival")))
			if err != nil {
				return manifest, err
			}
			if len(manifestYaml.Resources()) == 0 {
				return manifest, fmt.Errorf("The following file is not valid, since it does not contain any resource: %s", fileLink)
			}
			fmt.Println("appending", fileLink)
			fmt.Println("I found resources")
			fmt.Println(len(manifestYaml.Resources()))
			if index == 0 {
				manifest = manifestYaml
			} else {
				manifest = manifest.Append(manifest)
			}
		}

		fmt.Println("final resources")
		fmt.Println(len(manifest.Resources()))
	}

	if len(manifest.Resources()) == 0 {
		return manifest, fmt.Errorf("unable to find the manifest for the Knative version %s", version)
	}

	return manifest, nil
}
