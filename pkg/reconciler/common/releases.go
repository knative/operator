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
	"strconv"
	"strings"

	mf "github.com/manifestival/manifestival"
	"golang.org/x/mod/semver"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const (
	// KoEnvKey is the key of the environment variable to specify the path to the ko data directory
	KoEnvKey = "KO_DATA_PATH"

	// URLLinkTemplate is the template of the URL used to access the released artifacts of Knative
	URLLinkTemplate = "https://github.com/knative/%s/releases/download/v%s/%s"
)

var (
	cache = map[string]mf.Manifest{}

	// ServingYamlNames is the string array containing all the artifacts to download for Knative serving
	ServingYamlNames = []string{"serving-crds.yaml", "serving-core.yaml", "serving-hpa.yaml", "serving-storage-version-migration.yaml"}

	// EventingYamlNames is the string array containing all the artifacts to download for Knative eventing
	EventingYamlNames = []string{"eventing.yaml", "upgrade-to-%s.yaml", "storage-version-migration-%s.yaml"}
)

// TargetVersion returns the version of the manifest to be installed
// per the spec in the component. If spec.version is empty, the latest
// version known to the operator is returned.
func TargetVersion(instance v1alpha1.KComponent) string {
	target := instance.GetSpec().GetVersion()
	if target == "" {
		return latestRelease(instance)
	}
	return target
}

// TargetManifest returns the manifest for the TargetVersion
func TargetManifest(instance v1alpha1.KComponent) (mf.Manifest, error) {
	return fetch(manifestPath(TargetVersion(instance), instance))
}

// InstalledManifest returns the version currently installed, which is
// harder than it sounds, since status.version isn't set until the
// target version is successfully installed, which can take some time.
// So we return the target manifest if status.version is empty.
func InstalledManifest(instance v1alpha1.KComponent) (mf.Manifest, error) {
	current := instance.GetStatus().GetVersion()
	if current != "" {
		return fetch(manifestPath(current, instance))
	}
	return TargetManifest(instance)
}

// IsUpDowngradeEligible returns the bool indicate whether the installed manifest is able to upgrade or downgrade to
// the target manifest.
func IsUpDowngradeEligible(instance v1alpha1.KComponent) bool {
	current := instance.GetStatus().GetVersion()
	// If there is no manifest installed, return true, because the target manifest is able to install.
	if current == "" {
		return true
	}
	current = sanitizeSemver(current)
	target := sanitizeSemver(TargetVersion(instance))

	currentMajor := semver.Major(current)
	targetMajor := semver.Major(target)
	if currentMajor != targetMajor {
		// All the official releases of Knative are under the same Major version number. If target and current versions
		// are different in terms of major version, upgrade or downgrade is not supported.
		// TODO We need to deal with the the case of bumping major version later.
		return false
	}

	currentMinor, err := strconv.Atoi(strings.Split(current, ".")[1])
	if err != nil {
		return false
	}

	targetMinor, err := strconv.Atoi(strings.Split(target, ".")[1])
	if err != nil {
		return false
	}

	// If the diff between minor versions are less than 2, return true.
	if abs(currentMinor-targetMinor) < 2 {
		return true
	}

	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func fetch(path string) (mf.Manifest, error) {
	if m, ok := cache[path]; ok {
		return m, nil
	}
	result, err := mf.NewManifest(path)
	if err == nil {
		cache[path] = result
	}
	return result, err
}

func componentDir(instance v1alpha1.KComponent) string {
	koDataDir := os.Getenv(KoEnvKey)
	switch instance.(type) {
	case *v1alpha1.KnativeServing:
		return filepath.Join(koDataDir, "knative-serving")
	case *v1alpha1.KnativeEventing:
		return filepath.Join(koDataDir, "knative-eventing")
	}
	return ""
}

func componentURL(version string, instance v1alpha1.KComponent) string {
	component := "serving"
	urlList := []string{}
	switch instance.(type) {
	case *v1alpha1.KnativeServing:
		urlList = ServingYamlNames
	case *v1alpha1.KnativeEventing:
		component = "eventing"
		for _, value := range EventingYamlNames {
			if strings.Contains(value, "%s") {
				// Some eventing artifacts' names always contain the version with the patch number equal to 0
				value = fmt.Sprintf(value, fmt.Sprintf("%s.0", semver.MajorMinor(sanitizeSemver(version))))
			}
			urlList = append(urlList, value)
		}
	}

	// Create the comma-separated string as the URL to retrieve the manifest
	if len(urlList) == 0 {
		return ""
	}

	result := fmt.Sprintf(URLLinkTemplate, component,
		version, urlList[0])

	for i := 1; i < len(urlList); i++ {
		result = fmt.Sprintf("%s,"+URLLinkTemplate, result, component,
			version, urlList[i])
	}

	return result
}

func manifestPath(version string, instance v1alpha1.KComponent) string {
	localPath := filepath.Join(componentDir(instance), version)
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		return localPath
	}

	if !semver.IsValid(sanitizeSemver(version)) {
		return ""
	}

	return componentURL(version, instance)
}

// sanitizeSemver always adds `v` in front of the version.
// x.y.z is the standard format we use as the semantic version for Knative. The letter `v` is added for
// comparison purpose.
func sanitizeSemver(version string) string {
	return fmt.Sprintf("v%s", version)
}

// allReleases returns the all the available release versions
// available under kodata directory for Knative component.
func allReleases(instance v1alpha1.KComponent) ([]string, error) {
	// List all the directories available under kodata
	pathname := componentDir(instance)
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
		return nil, fmt.Errorf("unable to find any version number for %v", instance)
	}

	// This function makes sure the versions are sorted in a descending order.
	sort.Slice(releaseTags, func(i, j int) bool {
		// The index i is the one after the index j. If i is more recent than j, return true to swap.
		return semver.Compare(sanitizeSemver(releaseTags[i]), sanitizeSemver(releaseTags[j])) == 1
	})

	return releaseTags, nil
}

// latestRelease returns the latest release tag available under kodata directory for Knative component.
func latestRelease(instance v1alpha1.KComponent) string {
	vers, err := allReleases(instance)
	if err != nil {
		panic(err)
	}
	// The versions are in a descending order, so the first one will be the latest version.
	return vers[0]
}
