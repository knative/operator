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

package packages

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestPackage_String(t *testing.T) {
	p := Package{
		Name: "test",
		Primary: Source{
			GitHub: GitHubSource{
				Repo: "octocat/ignored",
			},
		},
	}

	want := "test"
	if p.String() != want {
		t.Errorf("Expected %q, got %q", want, p.String())
	}
}

func TestSource_String(t *testing.T) {
	s := Source{
		AssetFilter: AssetFilter{
			IncludeArtifacts: []string{".*good.*"},
			ExcludeArtifacts: []string{".*bad.*", ".*notgood.*"},
		},
		GitHub: GitHubSource{
			Repo: "octocat/repo",
		},
		Overrides: map[string]AssetFilter{
			"v0.2": {
				IncludeArtifacts: []string{".*good.*"},
				ExcludeArtifacts: []string{".*notgood.*"},
			},
		},
	}

	want := "octocat/repo"

	if s.String() != want {
		t.Errorf("Expected %q, got %q", want, s.String())
	}
}

func TestSource_OrgRepo(t *testing.T) {
	tests := []struct {
		path string
		org  string
		repo string
	}{
		{"knative/test", "knative", "test"},
		{"knative-sandbox/test-more", "knative-sandbox", "test-more"},
		{"unexpected/extra/slash", "unexpected", "extra/slash"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			source := Source{GitHub: GitHubSource{Repo: tt.path}}
			org, repo := source.OrgRepo()
			if org != tt.org {
				t.Errorf("Source.OrgRepo() got = %s, want %s", org, tt.org)
			}
			if repo != tt.repo {
				t.Errorf("Source.OrgRepo() got1 = %s, want %s", repo, tt.repo)
			}
		})
	}
}

func TestAssetFilter_Accept(t *testing.T) {

	input := []Asset{
		{Name: "foo.yaml"},
		{Name: "foo-extras.yaml"},
		{Name: "bar.yaml"},
		{Name: "bad.yaml"},
	}
	tests := []struct {
		name string
		af   AssetFilter
		want []Asset
	}{
		{
			name: "No Include",
			af: AssetFilter{
				IncludeArtifacts: []string{},
			},
			want: input,
		},
		{
			name: "Include all",
			af: AssetFilter{
				IncludeArtifacts: []string{".*.yaml"},
			},
			want: input,
		},
		{
			name: "F* only",
			af: AssetFilter{
				IncludeArtifacts: []string{"f.*.yaml"},
			},
			want: []Asset{
				{Name: "foo.yaml"},
				{Name: "foo-extras.yaml"},
			},
		},
		{
			name: "No bad",
			af: AssetFilter{
				IncludeArtifacts: []string{".*.yaml"},
				ExcludeArtifacts: []string{"bad.yaml"},
			},
			want: []Asset{
				{Name: "foo.yaml"},
				{Name: "foo-extras.yaml"},
				{Name: "bar.yaml"},
			},
		},
		{
			name: "F-rename",
			af: AssetFilter{
				Rename: map[string]string{
					"foo.yaml":        "test",
					"foo-extras.yaml": "test",
					"nobody":          "nothing",
				},
			},
			want: []Asset{
				{Name: "test"},
				{Name: "test"},
				{Name: "bar.yaml"},
				{Name: "bad.yaml"},
			},
		},
	}

	ignore := cmpopts.IgnoreUnexported(Asset{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make([]Asset, 0, len(input))
			for _, a := range input {
				if name := tt.af.Accept(a.Name); name != "" {
					item := a
					item.Name = name
					got = append(got, item)
				}
			}

			for i := range tt.want {
				if len(got) <= i {
					t.Errorf("Missing element %d. Wanted %v", i, tt.want[i])
					continue
				}
				if diff := cmp.Diff(tt.want[i], got[i], ignore); diff != "" {
					t.Errorf("Unexpected item at %d (-want +got): %s", i, diff)
				}
			}

			if len(tt.want) < len(got) {
				t.Errorf("Got wrong number of elements: want %d, got %d", len(tt.want), len(got))
			}
		})
	}
}

func TestSource_Accept(t *testing.T) {
	assets := []Asset{
		{Name: "foo.yaml"},
		{Name: "bar.yaml"},
		{Name: "bad.yaml"},
	}
	s := Source{
		AssetFilter: AssetFilter{
			ExcludeArtifacts: []string{"bad.yaml"},
		},
		Overrides: map[string]AssetFilter{
			"v0.1": {
				IncludeArtifacts: []string{"foo.yaml"},
			},
			"v0.1.3": {
				IncludeArtifacts: []string{"foo.yaml", "okay.yaml"},
				Rename: map[string]string{
					"bad.yaml": "okay.yaml",
				},
			},
			"v0.2.2": {
				IncludeArtifacts: []string{".*.yaml"},
			},
		},
	}

	tests := []struct {
		version string
		want    []Asset
	}{
		// TODO: Add test cases.
		{
			version: "v0.5.0",
			want: []Asset{
				{Name: "foo.yaml"},
				{Name: "bar.yaml"},
			},
		},
		{
			version: "v0.2.0",
			want: []Asset{
				{Name: "foo.yaml"},
				{Name: "bar.yaml"},
			},
		},
		{
			version: "v0.2.2",
			want: []Asset{
				{Name: "foo.yaml"},
				{Name: "bar.yaml"},
				{Name: "bad.yaml"},
			},
		},
		{
			version: "v0.1.2",
			want: []Asset{
				{Name: "foo.yaml"},
			},
		},
		{
			version: "v0.1.3",
			want: []Asset{
				{Name: "foo.yaml"},
				{Name: "okay.yaml"},
			},
		},
	}

	ignore := cmpopts.IgnoreUnexported(Asset{})
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := make([]Asset, 0, len(assets))
			f := s.Accept(tt.version)

			for _, a := range assets {
				if name := f(a.Name); name != "" {
					item := a
					item.Name = name
					got = append(got, item)
				}
			}

			for i := range tt.want {
				if len(got) <= i {
					t.Errorf("Missing element %d. Wanted %v", i, tt.want[i])
					continue
				}
				if diff := cmp.Diff(tt.want[i], got[i], ignore); diff != "" {
					t.Errorf("Unexpected item at %d (-want +got): %s", i, diff)
				}
			}

			if len(tt.want) < len(got) {
				t.Errorf("Got wrong number of elements: want %d, got %d", len(tt.want), len(got))
			}
		})
	}
}
