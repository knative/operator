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
	// VersionVariable is a string, which can be replaced with the value of spec.version
	VersionVariable = "${VERSION}"
	// COMMA is the character comma
	COMMA = ","
	// LATEST_VERSION is the special version Knative Operator support, besides all semantic versions of Knative.
	LATEST_VERSION = "latest"
)

var cache = map[string]mf.Manifest{}

// TargetVersion returns the version of the manifest to be installed
// per the spec in the component. If spec.version is empty, the latest
// version known to the operator is returned.
func TargetVersion(instance v1alpha1.KComponent) string {
	version := instance.GetSpec().GetVersion()
	if strings.EqualFold(version, LATEST_VERSION) {
		return getLatestRelease(instance, version)
	}

	if len(instance.GetSpec().GetManifests()) == 0 {
		if version == "" {
			return latestRelease(instance)
		}

		if SanitizeSemver(version) == semver.MajorMinor(SanitizeSemver(version)) {
			return getLatestRelease(instance, version)
		}
	}

	return version
}

// TargetManifest returns the default manifest for the TargetVersion or the manifest for the TargetVersion specified
// with spec.manifests
func TargetManifest(instance v1alpha1.KComponent) (mf.Manifest, error) {
	manifestsPath := targetManifestPath(instance)
	if len(instance.GetSpec().GetManifests()) == 0 {
		getManifestWithVersionValidation(manifestsPath, instance, FetchManifest)
	}
	return getManifestWithVersionValidation(manifestsPath, instance, fetchManifestFromPath)
}

// TargetAdditionalManifest returns the manifest for the TargetVersion specified with spec.additionalManifests.
func TargetAdditionalManifest(instance v1alpha1.KComponent) (mf.Manifest, error) {
	additionalManifestsPath := additionalManifestPath(instance)
	if additionalManifestsPath == "" {
		return mf.Manifest{}, nil
	}
	return getManifestWithVersionValidation(additionalManifestsPath, instance, fetchManifestFromPath)
}

// InstalledManifest returns the version currently installed, which is
// harder than it sounds, since status.version isn't set until the
// target version is successfully installed, which can take some time.
// So we return the target manifest if status.version is empty.
func InstalledManifest(instance v1alpha1.KComponent) (mf.Manifest, error) {
	current := instance.GetStatus().GetVersion()
	if len(instance.GetStatus().GetManifests()) == 0 && current == "" {
		return TargetManifest(instance)
	}
	// If status.manifests is not empty, get the manifests from the cache if available, and get them from
	// the path if not available in the cache.
	// Read the path one by one, in order to leverage the cache, because the whole comma-separated path is
	// not saved in the cache, but each path is saved as the key of the cache.
	paths := installedManifestPath(current, instance)
	if len(paths) == 0 {
		return mf.Manifest{}, nil
	}
	manifest, err := FetchManifest(paths[0])
	if err != nil {
		return manifest, err
	}
	for i := 1; i < len(paths); i++ {
		m, er := FetchManifest(paths[i])
		if er != nil {
			return manifest, er
		}
		manifest = manifest.Append(m)
	}
	return manifest, err
}

// IsVersionValidMigrationEligible returns the bool indicate whether the target version is valid and the installed
// manifest is able to upgrade or downgrade to the target manifest.
func IsVersionValidMigrationEligible(instance v1alpha1.KComponent) error {
	var err error
	targetVersion := TargetVersion(instance)
	if targetVersion == LATEST_VERSION {
		return nil
	}
	target := SanitizeSemver(targetVersion)
	if !semver.IsValid(target) {
		return fmt.Errorf("target version %v is not in a valid semantic versioning format.", target)
	}

	if len(strings.Split(target, ".")) < 2 {
		return fmt.Errorf("target version %v should at least include the major and minor numbers.", target)
	}

	current := instance.GetStatus().GetVersion()
	// If there is no manifest installed, return nil, because the target manifest is able to install.
	// If the installed manifest is versioned with latest, we allow any version to upgrade to or from it.
	if current == "" || current == LATEST_VERSION {
		return nil
	}

	current = SanitizeSemver(current)
	currentMajor := semver.Major(current)
	targetMajor := semver.Major(target)
	if currentMajor != targetMajor {
		// All the official releases of Knative are under the same Major version number. If target and current versions
		// are different in terms of major version, upgrade or downgrade is not supported.
		// TODO We need to deal with the the case of bumping major version later.
		return fmt.Errorf("not supported to upgrade or downgrade across the MAJOR version. The "+
			"installed KnativeServing version is %v.", current)
	}

	currentMinor, err := strconv.Atoi(strings.Split(current, ".")[1])
	if err != nil {
		return fmt.Errorf("minor number of the current version %v should be an integer.", current)
	}
	targetMinor, err := strconv.Atoi(strings.Split(target, ".")[1])
	if err != nil {
		return fmt.Errorf("minor number of the target version %v should be an integer.", target)
	}

	// If the diff between minor versions are less than 2, return nil.
	if abs(currentMinor-targetMinor) < 2 {
		return nil
	}

	return fmt.Errorf("not supported to upgrade or downgrade across multiple MINOR versions. The "+
		"installed KnativeServing version is %v.", current)
}

func getVersionKey(instance v1alpha1.KComponent) string {
	switch instance.(type) {
	case *v1alpha1.KnativeServing:
		return "serving.knative.dev/release"
	case *v1alpha1.KnativeEventing:
		return "eventing.knative.dev/release"
	}
	return ""
}

type manifestFetcher func(string) (mf.Manifest, error)

func getManifestWithVersionValidation(manifestsPath string, instance v1alpha1.KComponent, fn manifestFetcher) (mf.Manifest, error) {
	version := TargetVersion(instance)
	manifests, err := fn(manifestsPath)
	if err != nil {
		if len(instance.GetSpec().GetManifests()) == 0 && len(instance.GetSpec().GetAdditionalManifests()) == 0 {
			// If we cannot access the manifests, there is no need to check whether the versions match.
			// If both spec.manifests and spec.additionalManifests are empty, there is no need to check whether the versions
			// match.
			return manifests, fmt.Errorf("The manifests of the target version %v are not available to this release.",
				instance.GetSpec().GetVersion())
		}
		return manifests, err
	}

	if len(manifests.Resources()) == 0 {
		// If we cannot find any resources in the manifests, we need to return an error.
		return manifests, fmt.Errorf("There is no resource available in the target manifests %s.", manifestsPath)
	}

	if version == "" || version == LATEST_VERSION {
		// If target version is empty or equal to the special latest version, there is no need to check whether
		// the versions match.
		return manifests, nil
	}

	targetVersion := SanitizeSemver(version)
	key := getVersionKey(instance)
	for _, u := range manifests.Resources() {
		// Check the labels of the resources one by one to see if the version matches the target version in terms of
		// major.minor.
		manifestVersion := u.GetLabels()[key]
		if manifestVersion != "" && semver.MajorMinor(targetVersion) != semver.MajorMinor(manifestVersion) {
			return mf.Manifest{}, fmt.Errorf("The version of the manifests %s does not match the target "+
				"version of the operator CR %s. The resource name is %s.", manifestVersion, targetVersion, u.GetName())
		}
	}

	return manifests, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// FetchManifest returns the manifest by either getting it from the cache, or reading them from the path.
// The manifest is saved in the cache, if it is not available.
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

// fetchManifestFromPath returns the manifest by reading them from the path, and saves them in the cache.
func fetchManifestFromPath(path string) (mf.Manifest, error) {
	result, err := mf.NewManifest(path)
	if err == nil {
		cache[path] = result
	}
	return result, err
}

// ClearCache removes all the records saved in the cache.
func ClearCache() {
	cache = map[string]mf.Manifest{}
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

func additionalManifestPath(instance v1alpha1.KComponent) string {
	// Create the comma-separated string for URLs in spec.additionalManifests
	addManifests := instance.GetSpec().GetAdditionalManifests()
	urls := make([]string, 0, len(addManifests))
	for _, manifest := range addManifests {
		url := strings.ReplaceAll(manifest.Url, VersionVariable, TargetVersion(instance))
		urls = append(urls, url)
	}
	return strings.Join(urls, COMMA)
}

func targetManifestPath(instance v1alpha1.KComponent) string {
	version := TargetVersion(instance)
	manifests := instance.GetSpec().GetManifests()
	// Create the comma-separated string as the URL to retrieve the manifest
	urls := make([]string, 0, len(manifests))
	for _, manifest := range manifests {
		url := strings.ReplaceAll(manifest.Url, VersionVariable, version)
		urls = append(urls, url)
	}

	manifestPath := strings.Join(urls, COMMA)
	// If spec.manifests is empty, add the local path
	if manifestPath == "" {
		manifestPath = filepath.Join(componentDir(instance), version)
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			return ""
		}
	}
	return manifestPath
}

func targetManifestPathArray(instance v1alpha1.KComponent) []string {
	targetMPath := targetManifestPath(instance)
	manifestPaths := []string{targetMPath}
	if len(instance.GetSpec().GetAdditionalManifests()) > 0 {
		// If spec.additionalManifests is not empty, we append it to the target path.
		additionalMPath := additionalManifestPath(instance)
		manifestPaths = append(manifestPaths, additionalMPath)
	}
	return manifestPaths
}

func installedManifestPath(version string, instance v1alpha1.KComponent) []string {
	if manifests := instance.GetStatus().GetManifests(); len(manifests) != 0 {
		return manifests
	}

	localPath := filepath.Join(componentDir(instance), version)
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		return []string{localPath}
	}
	return []string{}
}

// SanitizeSemver always adds `v` in front of the version.
// x.y.z is the standard format we use as the semantic version for Knative. The letter `v` is added for
// comparison purpose.
func SanitizeSemver(version string) string {
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
		return semver.Compare(SanitizeSemver(releaseTags[i]), SanitizeSemver(releaseTags[j])) == 1
	})

	return releaseTags, nil
}

// latestRelease returns the latest release tag available under kodata directory for Knative component.
func latestRelease(instance v1alpha1.KComponent) string {
	return getLatestRelease(instance, "")
}

// getLatestRelease returns the latest release tag available under kodata directory for Knative component
// based on spec.version.
func getLatestRelease(instance v1alpha1.KComponent, version string) string {
	// The versions are in a descending order, so the first one will be the latest version.
	vers, err := allReleases(instance)
	if err != nil {
		panic(err)
	}

	if version == "" {
		return vers[0]
	}

	if strings.EqualFold(version, LATEST_VERSION) {
		// If spec.version is set to latest, look up if the directory latest is available.
		// If not, return the newest available version instead.
		for _, val := range vers {
			if val == version {
				return val
			}
		}
		return vers[0]
	}

	for _, val := range vers {
		if strings.HasPrefix(val, version) &&
			semver.MajorMinor(SanitizeSemver(val)) == semver.MajorMinor(SanitizeSemver(version)) {
			// If spec.version is set in the format of major.minor, we return the latest version matching
			// spec.version.
			return val
		}
	}
	return version
}
