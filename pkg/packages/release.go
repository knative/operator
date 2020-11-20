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

// Package packages provides abstract tools for managing a packaged release.
package packages

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

// Asset provides an abstract interface for describing a resource which should be stored on the disk.
type Asset struct {
	Name string
	URL  string
}

// Release provides an interface for a release which contains multiple assets at the same release (TagName)
type Release struct {
	Org     string
	Repo    string
	TagName string
	Assets  []Asset
}

// Less provides a method for implementing `sort.Slice` to ensure that assets are applied in the correct order.
func (a Asset) Less(b Asset) bool {
	if strings.HasSuffix(a.Name, "-crds.yaml") {
		return true
	}
	if strings.HasSuffix(b.Name, "-crds.yaml") {
		return false
	}
	if strings.HasSuffix(a.Name, "-post-install-jobs.yaml") {
		return true
	}
	if strings.HasSuffix(b.Name, "-post-install-jobs.yaml") {
		return false
	}
	return a.Name < b.Name
}

// Less provides a sort on Releases by TagName.
func (r Release) Less(b Release) bool {
	return semver.Compare(r.TagName, b.TagName) < 0
}

// String implements `fmt.Stringer`.
func (r Release) String() string {
	return fmt.Sprintf("%s/%s %s", r.Org, r.Repo, r.TagName)
}

// SortAssets ensures that the assets in the resource are in correct order for application.
func (r *Release) SortAssets() {
	sort.Slice(r.Assets, func(i, j int) bool { return r.Assets[i].Less(r.Assets[j]) })
}

// FilterAssets does an IN-PLACE removal of assets from the selected release which do not match the `retain` filter.
func (r *Release) FilterAssets(retain func(string) bool) {
	raw := r.Assets
	r.Assets = make([]Asset, 0, len(r.Assets))
	for _, asset := range raw {
		if retain(asset.Name) {
			r.Assets = append(r.Assets, asset)
		}
	}
}

// HandleRelease processes the files for a given release of the specified
// Package.
//
// NOTE: This does not currently handle Additional assets, and ends up modifying `r` in a way that it probably shouldn't.
func HandleRelease(ctx context.Context, client *http.Client, p Package, r Release) error {
	shortName := strings.TrimPrefix(r.TagName, "v")
	path := filepath.Join("cmd", "operator", "kodata", p.Name, shortName)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	// TODO: make a copy of r's assets to avoid modifying the global cache.

	r.FilterAssets(p.Primary.Accept)
	r.SortAssets()

	// Download assets and store them.
	for i, asset := range r.Assets {
		fileName := fmt.Sprintf("%d-%s", i+1, asset.Name)
		file, err := os.OpenFile(filepath.Join(path, fileName), os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Unable to open %s: %w", fileName, err)
		}
		defer file.Close()
		if _, err := file.WriteString("# " + asset.URL + "\n\n"); err != nil {
			return err
		}
		log.Print(asset.URL)
		fetch, err := client.Get(asset.URL)
		if err != nil {
			return fmt.Errorf("Unable to fetch %s: %w", fileName, err)
		}
		defer fetch.Body.Close()
		_, err = io.Copy(file, fetch.Body)
		if err != nil {
			return fmt.Errorf("Unable to write to %s: %w", fileName, err)
		}
	}
	return nil
}

// LastN selects the last N minor releases (including all patch releases) for a
// given sequence of releases, which need not be sorted.
func LastN(minors int, allReleases []Release) []Release {
	retval := make([]Release, len(allReleases))

	copy(retval, allReleases)
	sort.Slice(retval, func(i, j int) bool {
		// Sort larger items earlier
		return !retval[i].Less(retval[j])
	})

	previous := semver.MajorMinor(retval[0].TagName)
	for i, r := range retval {
		if semver.MajorMinor(r.TagName) == previous {
			continue // Only count/act if the minor release changes
		}
		previous = semver.MajorMinor(r.TagName)
		minors--
		if minors == 0 {
			retval = retval[:i]
			break
		}
	}

	return retval
}
