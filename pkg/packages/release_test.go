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
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	orderedAssets = []Asset{
		{Name: "foo-pre-install-jobs.yaml"},
		{Name: "zoo-pre-install-jobs.yaml"},
		{Name: "bar-pre-install-jobs.yaml", secondary: true},
		{Name: "foo-crds.yaml"},
		{Name: "foo.yaml"},
		{Name: "roles.yaml"},
		{Name: "bar.yaml", secondary: true},
		{Name: "baz.yaml", secondary: true},
		{Name: "foo-sugar-controller.yaml"},
		{Name: "foo-post-install-jobs.yaml"},
	}

	orderedReleases = []Release{
		{TagName: "v0.1"},
		{TagName: "v0.2"},
		{TagName: "v0.2.1"},
		{TagName: "v0.2.2"},
		{TagName: "v0.2.8"},
		{TagName: "v0.3"},
		{TagName: "v0.3.1"},
		{TagName: "v0.3.2"},
		{TagName: "v0.4"},
		{TagName: "v0.5"},
		{TagName: "v0.5.1"},
	}
)

func TestAsset_Less(t *testing.T) {
	for i, first := range orderedAssets {
		for _, second := range orderedAssets[i+1:] {
			if !first.Less(second) {
				t.Errorf("Failed expectation %v < %v", first, second)
			}
			if second.Less(first) {
				t.Errorf("Failed expectation %v < %v", second, first)
			}
		}
	}
}

func TestRelease_Less(t *testing.T) {
	for i, first := range orderedReleases {
		for _, second := range orderedReleases[i+1:] {
			if !first.Less(second) {
				t.Errorf("Failed expectation %v < %v", first, second)
			}
			if second.Less(first) {
				t.Errorf("Failed expectation %v < %v", second, first)
			}
		}
	}
}

func TestRelease_String(t *testing.T) {
	rel := Release{
		Org:     "example",
		Repo:    "test",
		TagName: "v0.3.3",
		Created: time.Now(),
		Assets: []Asset{
			{
				Name:      "Ignore",
				URL:       "Unneeded",
				secondary: false,
			},
		},
	}
	want := "example/test v0.3.3"
	if got := rel.String(); got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

func Test_AssetSort(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	shuffled := make(assetList, len(orderedAssets))
	copy(shuffled, orderedAssets)

	for {
		rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if !sort.IsSorted(shuffled) {
			break
		}
	}

	sort.Sort(shuffled)

	for i := range shuffled {
		if orderedAssets[i] != shuffled[i] {
			t.Errorf("Sort mismatched at %d: got = %v, want = %v", i, shuffled[i], orderedAssets[i])
		}
	}
}

func Test_ReleaseSort(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	shuffled := make(releaseList, len(orderedReleases))
	copy(shuffled, orderedReleases)

	for {
		rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if !sort.IsSorted(shuffled) {
			break
		}
	}

	sort.Sort(shuffled)

	for i := range shuffled {
		// Releases sort newest-to-oldest, while Release.Less is just a semver
		// sort. Reverse the order of releases on the fly.
		if diff := cmp.Diff(orderedReleases[len(orderedReleases)-i-1], shuffled[i]); diff != "" {
			t.Errorf("Sort mismatched at %d (-want, +got): %s", i, diff)
		}
	}
}

func Test_assetList_FilterAssets(t *testing.T) {
	assets := assetList{
		{Name: "test.yaml"},
		{Name: "good.yaml"},
		{Name: "good-for-you.yaml"},
		{Name: "bad.yaml"},
	}
	tests := []struct {
		name string
		f    func(string) string
		want assetList
	}{
		{
			name: "Pass-through",
			f: func(s string) string {
				return s
			},
			want: assets,
		},
		{
			name: "Remove",
			f: func(string) string {
				return ""
			},
			want: []Asset{},
		},
		{
			name: "Keep good",
			f: func(s string) string {
				if strings.HasPrefix(s, "good") {
					return s
				}
				return ""
			},
			want: []Asset{{Name: "good.yaml"}, {Name: "good-for-you.yaml"}},
		},
		{
			name: "Rename bad",
			f: func(s string) string {
				return strings.Replace(s, "bad", "okay", -1)
			},
			want: []Asset{
				{Name: "test.yaml"},
				{Name: "good.yaml"},
				{Name: "good-for-you.yaml"},
				{Name: "okay.yaml"},
			},
		},
	}
	ignore := cmpopts.IgnoreUnexported(Asset{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, assets.FilterAssets(tt.f), ignore); diff != "" {
				t.Errorf("assetList.FilterAssets(%v) (-want, +got): %s", assets, diff)
			}
		})
	}
}

func TestLastN(t *testing.T) {
	releases := make(releaseList, len(orderedReleases))
	copy(releases, orderedReleases)
	// Put in most-recent-to-oldest order.
	sort.Sort(releases)

	tests := []struct {
		count  int
		oldest string
	}{
		{5, "v0.1"},
		{4, "v0.2"},
		{3, "v0.3"},
		{2, "v0.4"},
		{1, "v0.5"},
		{0, "v0.1"},
		{8, "v0.1"},
	}
	ignore := cmpopts.IgnoreUnexported(Asset{})
	for _, tt := range tests {
		t.Run(tt.oldest, func(t *testing.T) {
			subset := LastN(tt.count, releases)

			for i := range subset {
				if diff := cmp.Diff(releases[i], subset[i], ignore); diff != "" {
					t.Errorf("Unexpect release at %d (-want +got): %s", i, diff)
				}
			}

			last := len(subset) - 1
			if subset[last].TagName != tt.oldest {
				t.Errorf("Incorrect lastN(%d) release, want = %s, got = %s", tt.count, tt.oldest, subset[last].TagName)
			}
		})
	}
}
