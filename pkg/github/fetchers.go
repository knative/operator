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

package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v32/github"
	"knative.dev/operator/pkg/packages"
)

// GetReleases returns all the releases in the specified org and repo
func GetReleases(ctx context.Context, client *github.Client, org string, repo string) ([]packages.Release, error) {
	opt := &github.ListOptions{PerPage: 100}

	retval := []packages.Release{}

	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, org, repo, opt)
		if err != nil || resp.StatusCode != 200 {
			if err == nil {
				err = fmt.Errorf("Got HTTP %d: %s", resp.StatusCode, resp.Status)
			}
			return nil, err
		}
		for _, release := range releases {
			retval = append(retval, makeRelease(org, repo, release))
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return retval, nil
}

func makeRelease(org string, repo string, gh *github.RepositoryRelease) packages.Release {
	retval := packages.Release{
		Org:     org,
		Repo:    repo,
		TagName: gh.GetTagName(),
		Assets:  make([]packages.Asset, 0, len(gh.Assets)),
	}
	for _, a := range gh.Assets {
		retval.Assets = append(retval.Assets, packages.Asset{Name: a.GetName(), URL: a.GetBrowserDownloadURL()})
	}
	return retval
}
