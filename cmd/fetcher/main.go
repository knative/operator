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
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	ghclient "github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
	"knative.dev/operator/pkg/blob"
	"knative.dev/operator/pkg/github"
	"knative.dev/operator/pkg/packages"
)

func main() {
	cfg, err := packages.ReadConfig("cmd/fetcher/kodata/config.yaml")
	if err != nil {
		log.Print("Unable to read config: ", err)
		os.Exit(2)
	}

	ctx := context.Background()
	client := getClient(ctx)
	ghClient := ghclient.NewClient(client)
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
				log.Printf("Unable to fetch %s: %v", release, err)
			}
			log.Printf("Wrote %s ==> %s", v.String(), release.String())
		}
	}
}

func getClient(ctx context.Context) *http.Client {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return nil
	}
	staticToken := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")})
	return oauth2.NewClient(ctx, staticToken)
}

func ensureRepo(ctx context.Context, known map[string][]packages.Release, client *ghclient.Client, src packages.Source) error {
	if known[src.String()] != nil {
		return nil
	}
	if src.GitHub != (packages.GitHubSource{}) {
		if client == nil {
			return fmt.Errorf("must set $GITHUB_TOKEN to use github sources.")
		}
		owner, repo := src.OrgRepo()
		releases, err := github.GetReleases(ctx, client, owner, repo)
		if err != nil {
			return err
		}
		known[src.String()] = releases
		return nil
	}
	if src.S3 != (packages.S3Source{}) {
		releases, err := blob.GetReleases(ctx, &http.Client{}, src.S3)
		if err != nil {
			return err
		}
		known[src.String()] = releases
		return nil
	}
	return errors.New("Must specify one of S3 or GitHub")
}
