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
	"sort"

	semver "github.com/blang/semver/v4"
	"github.com/google/go-github/v31/github"
)

// GetLatestOnlinePatchRelease returns the latest patch release tag for org/repo repository, with a version number.
// If no release tag is found, it returns an empty string
func GetLatestOnlinePatchRelease(org, repo, version string) (string, error) {
	releaseTag := ""
	releaseTags, err := ListOnlineReleases(org, repo)

	if err != nil {
		return releaseTag, err
	}

	for _, onlineVersion := range releaseTags {
		// Check if the onlineVersion matches version, in terms of MAJOR and MINOR. We will return the latest
		// patch release version, based on the provided version. The releaseTags has been sorted in a descending
		// order, so the first one we find will be the latest patch release version.
		fmt.Println(onlineVersion)
		onlineSemVer, err := semver.Make(onlineVersion)
		if err != nil {
			return version, err
		}
		providedSemVer, err := semver.Make(version)
		if err != nil {
			return version, err
		}
		if onlineSemVer.Major == providedSemVer.Major && onlineSemVer.Minor == providedSemVer.Minor {
			return onlineVersion, nil
		}
	}

	return version, nil
}

// ListOnlineReleases returns an array of version available for org/repo repository in a SemVer-based descending order.
func ListOnlineReleases(org, repo string) ([]string, error) {
	releaseTags := []string{}
	client := github.NewClient(nil)
	opt := &github.ListOptions{}
	releases, _, err := client.Repositories.ListReleases(context.Background(), org, repo, opt)

	if err != nil {
		return releaseTags, err
	}

	if len(releases) == 0 {
		return releaseTags, nil
	}

	for _, val := range releases {
		releaseTag := val.GetTagName()
		// Remove the first letter v to get MAJOR.MINOR.PATCH
		release := string(releaseTag[1:])
		releaseTags = append(releaseTags, release)
	}

	sort.Slice(releaseTags, func(i, j int) bool {
		vI, err := semver.Make(releaseTags[i])
		if err != nil {
			return false
		}

		vJ, err := semver.Make(releaseTags[j])
		if err != nil {
			return false
		}

		// Sort the list of version numbers in a descending order.
		return vI.Compare(vJ) == 1
	})

	return releaseTags, nil
}
