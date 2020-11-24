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

// Package main is the main package for the fetcher. The fetcher knows how to
// collect a directory tree of release artifacts given a configuration file
// indicating the desired top-level packages.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	ghclient "github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
	"knative.dev/operator/pkg/github"
	"knative.dev/operator/pkg/packages"
)

var interested = map[string]struct{}{
	"knative/serving":  {},
	"knative/eventing": {},
}

func main() {
	cfg, err := packages.ReadConfig("cmd/fetcher/kodata/config.yaml")
	if err != nil {
		log.Printf("Unable to read config: %v", err)
		os.Exit(2)
	}

	ctx := context.Background()
	ghClient := ghclient.NewClient(getClient(ctx))
	repos := make(map[string][]packages.Release, len(cfg))

	for _, v := range cfg {
		if err := ensureRepo(ctx, repos, ghClient, v.Primary); err != nil {
			log.Printf("Unable to fetch %s: %v", v.Primary, err)
			os.Exit(2)
		}
		for _, s := range v.Additional {
			if err := ensureRepo(ctx, repos, ghClient, s); err != nil {
				log.Printf("Unable to fetch %s: %v", s, err)
				os.Exit(2)
			}
		}

		base := filepath.Join("cmd", "operator", "kodata", v.Name)
		if err := os.RemoveAll(base); err != nil && !os.IsNotExist(err) {
			log.Printf("Unable to remove directory %s: %v", base, err)
			os.Exit(3)
		}

		for _, release := range packages.LastN(4, repos[v.Primary.String()]) {
			if err := packages.HandleRelease(ctx, http.DefaultClient, *v, release, repos); err != nil {
				log.Printf("Unable to fetch %s, %v", release, err)
			}
			log.Printf("Wrote %s ==> %s", v.String(), release.String())
		}
	}
}

func getClient(ctx context.Context) *http.Client {
	staticToken := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")})
	return oauth2.NewClient(ctx, staticToken)
}

func ensureRepo(ctx context.Context, known map[string][]packages.Release, client *ghclient.Client, src packages.Source) error {
	if known[src.GitHub.Repo] != nil {
		return nil
	}
	owner, repo := src.OrgRepo()
	releases, err := github.GetReleases(ctx, client, owner, repo)
	if err != nil {
		return err
	}
	known[src.GitHub.Repo] = releases
	return nil
}
