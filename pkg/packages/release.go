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
	"time"

	"golang.org/x/mod/semver"
	"knative.dev/operator/pkg/reconciler/common"
)

// Asset provides an abstract interface for describing a resource which should be stored on the disk.
type Asset struct {
	Name      string
	URL       string
	secondary bool
}

// Release provides an interface for a release which contains multiple assets at the same release (TagName)
type Release struct {
	Org     string
	Repo    string
	TagName string
	Created time.Time
	Assets  assetList //[]Asset
}

var suffixOrder = map[string]int{
	// HACK for pre-install jobs, which are deprecated, because the job needs to
	// *complete*, not just be applied, before the next manifests can be
	// applied.
	"-pre-install-jobs.yaml": -6,
	"-crds.yaml":             -3,
	// HACK for eventing, which lists the sugar controller after the
	// channel/broker despite collating before.
	"-sugar-controller.yaml":  3,
	"-post-install-jobs.yaml": 6,
	"-post-install.yaml":      6,
}

// Less provides a method for implementing `sort.Slice` to ensure that assets
// are applied in the correct order.
func (a Asset) Less(b Asset) bool {
	aScore, bScore := 0, 0
	for suffix, score := range suffixOrder {
		if strings.HasSuffix(a.Name, suffix) {
			aScore = score
		}
		if strings.HasSuffix(b.Name, suffix) {
			bScore = score
		}
	}
	// Sort primary assets before secondary assets
	if a.secondary {
		aScore++
	}
	if b.secondary {
		bScore++
	}

	if aScore == bScore {
		return a.Name < b.Name
	}
	return aScore < bScore
}

// Less provides a sort on Releases by TagName.
func (r Release) Less(b Release) bool {
	return semver.Compare(r.TagName, b.TagName) < 0
}

// String implements `fmt.Stringer`.
func (r Release) String() string {
	return fmt.Sprintf("%s/%s %s", r.Org, r.Repo, r.TagName)
}

// assetList provides an interface for operating on a set of assets.
type assetList []Asset

// Len is part of `sort.Interface`.
func (al assetList) Len() int {
	return len(al)
}

// Less is part of `sort.Interface`.
func (al assetList) Less(i, j int) bool {
	return al[i].Less(al[j])
}

// Swap is part of `sort.Interface`.
func (al assetList) Swap(i, j int) {
	al[i], al[j] = al[j], al[i]
}

type releaseList []Release

// Len is part of `sort.Interface`.
func (rl releaseList) Len() int {
	return len(rl)
}

// Less is part of `sort.Interface`.  Note that this is actually a reversed
// sort, because we want newest releases towards the start of the list.
func (rl releaseList) Less(i, j int) bool {
	return !rl[i].Less(rl[j])
}

// Swap is part of `sort.Interface`.
func (rl releaseList) Swap(i, j int) {
	rl[i], rl[j] = rl[j], rl[i]
}

// FilterAssets retains only assets where `accept` returns a non-empty string.
// `accept` may return a *different* string in the case of assets which should
// be renamed.
func (al assetList) FilterAssets(accept func(string) string) assetList {
	retval := make([]Asset, 0, len(al))
	for _, asset := range al {
		if name := accept(asset.Name); name != "" {
			asset.Name = name
			retval = append(retval, asset)
		}
	}

	return retval
}

func CollectReleaseAssets(p Package, r Release, allReleases map[string][]Release) []Asset {
	assets := make(assetList, 0, len(r.Assets))
	assets = append(assets, r.Assets.FilterAssets(p.Primary.Accept(r.TagName))...)
	for _, src := range p.Additional {
		candidates := allReleases[src.String()]
		sort.Sort(releaseList(candidates))
		start, end := -1, len(candidates)
		for i, srcRelease := range candidates {
			// Collect matching minor versions
			comp := semver.Compare(semver.MajorMinor(r.TagName), semver.MajorMinor(srcRelease.TagName))
			if start == -1 && comp == 0 {
				start = i
			}
			if comp > 0 {
				end = i
				break
			}
		}
		candidates = candidates[start:end]
		timeMatch := len(candidates) - 1
		for i, srcRelease := range candidates {
			// TODO: more sophisticated alignment options, for example, always use latest matching minor.
			if r.Created.After(srcRelease.Created) {
				timeMatch = i
				break
			}
		}

		candidate := candidates[timeMatch]
		newAssets := candidate.Assets.FilterAssets(src.Accept(candidate.TagName))
		for i := range newAssets {
			newAssets[i].secondary = true
		}
		assets = append(assets, newAssets...)
		log.Printf("Using %s/%s with %s/%s", candidate.String(), candidate.TagName, r.String(), r.TagName)
	}
	sort.Sort(assets)
	return assets
}

// HandleRelease processes the files for a given release of the specified
// Package.
func HandleRelease(ctx context.Context, base string, client *http.Client, p Package, r Release, allReleases map[string][]Release) error {
	if p.Alternatives {
		return handleAlternatives(ctx, base, client, p, r, allReleases)
	}
	return handlePrimary(ctx, base, client, p, r, allReleases)
}

// handlePrimary handles the files for a primary-style package.
func handlePrimary(ctx context.Context, base string, client *http.Client, p Package, r Release, allReleases map[string][]Release) error {
	assets := CollectReleaseAssets(p, r, allReleases)

	shortName := strings.TrimPrefix(r.TagName, "v")
	path := filepath.Join(base, p.Name, shortName)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	// Download assets and store them.
	for i, asset := range assets {
		fileName := fmt.Sprintf("%d-%s", i+1, asset.Name)
		file, err := os.OpenFile(filepath.Join(path, fileName), os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Unable to open %s: %w", fileName, err)
		}
		defer file.Close()
		log.Print(asset.URL)
		fetch, err := client.Get(asset.URL)
		if err != nil {
			return fmt.Errorf("Unable to fetch %s: %w", asset.URL, err)
		}
		defer fetch.Body.Close()
		_, err = io.Copy(file, fetch.Body)
		if err != nil {
			return fmt.Errorf("Unable to write to %s: %w", fileName, err)
		}
	}
	return nil
}

func handleAlternatives(ctx context.Context, base string, client *http.Client, p Package, r Release, allReleases map[string][]Release) error {
	minor := semver.MajorMinor(r.TagName)
	if lm := latestMinor(minor, allReleases[p.Primary.String()]); lm.TagName != r.TagName {
		log.Printf("Skipping %q, %q is newer", r.TagName, lm.TagName)
		return nil
	}

	shortName := strings.TrimPrefix(minor, "v")
	path := filepath.Join(base, p.Name, shortName)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	for _, src := range p.Additional {
		candidates := allReleases[src.String()]
		resourcePath := path
		if src.EventingService != "" {
			resourcePath = filepath.Join(path, src.EventingService)
			err := os.MkdirAll(resourcePath, 0755)
			if err != nil {
				return err
			}
		}
		if src.IngressService != "" {
			resourcePath = filepath.Join(path, src.IngressService)
			err := os.MkdirAll(resourcePath, 0755)
			if err != nil {
				return err
			}
		}
		release := latestMinor(minor, candidates)
		// Download assets and concatenate them.
		assets := release.Assets.FilterAssets(src.Accept(release.TagName))
		for _, a := range assets {
			fileName := filepath.Join(resourcePath, a.Name)
			file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("Unable to open %s: %w", fileName, err)
			}
			defer file.Close()
			log.Print(a.URL)
			fetch, err := client.Get(a.URL)
			if err != nil {
				return fmt.Errorf("Unable to fetch %s: %w", a.URL, err)
			}
			defer fetch.Body.Close()
			_, err = io.Copy(file, fetch.Body)
			if err != nil {
				return fmt.Errorf("Unable to write to %s: %w", fileName, err)
			}
		}
	}
	return nil
}

// LastN selects the last N minor releases (including all patch releases) for a
// given sequence of releases, which need not be sorted.
func LastN(latestVersion string, minors int, allReleases []Release) []Release {
	retval := make(releaseList, len(allReleases))
	copy(retval, allReleases)
	sort.Sort(retval)

	if !strings.EqualFold(latestVersion, common.LATEST_VERSION) {
		startIndex := 0
		for i, r := range retval {
			if semver.MajorMinor(r.TagName) == latestVersion {
				startIndex = i
				break
			}
		}
		retval = retval[startIndex:]
	}

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

func latestMinor(minor string, choices []Release) Release {
	ret := Release{
		TagName: minor,
	}
	for _, release := range choices {
		if semver.Compare(minor, semver.MajorMinor(release.TagName)) == 0 {
			if semver.Compare(ret.TagName, release.TagName) <= 0 {
				ret = release
			}
		}
	}
	return ret
}
