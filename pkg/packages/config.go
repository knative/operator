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
	"errors"
	"os"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
	"sigs.k8s.io/yaml"
)

// Package represents a single deployable set of software artifacts, possibly
// composed of several repo-level releases.
type Package struct {
	// Name is a top-level directory name that the releases should be stored in.
	// This is collected from a map key in the configuration and is not directly
	// loaded from YAML.
	Name string `json:"-"`

	// If Alternatives is true, this will be considered an "alternatives"
	// collection, which contains single file alternatives for each Additional
	// item based on the latest minor (but not patch) versions of Primary.
	Alternatives bool

	// Primary is the primary source of release artifacts; collections of
	// release artifacts will be numbered based on the primary source's release
	// numbering scheme.
	Primary Source

	// Additional sources provide secondary artifacts which should be bundled
	// with the artifacts of the primary release. This can be useful (for
	// example) to select plugins which should be included with a base package.
	Additional []Source `json:",omitempty"`
}

// Source is represents the release artifacts of a given project or repo, which
// provides a sequence of semver-tagged artifact collections for an individual
// release.
type Source struct {
	AssetFilter `json:",inline"`
	// GitHub represents software released on GitHub using GitHub releases.
	GitHub GitHubSource `json:"github,omitempty"`
	// S3 represents software manifests stored in an blob storage service under
	// a specified prefix. The blob paths should end with "vX.Y.Z/<asset name>"
	S3 S3Source `json:"s3,omitempty"`
	// EventingService represents the name of the service for the eventing source
	EventingService string `json:"eventingService,omitempty"`

	// Overrides provides a mechanism for modifying include/exclude (and
	// possibly other settings) on a per-release or per-minor-version basis, to
	// allow fixing up discontinuities in release patterns.
	Overrides map[string]AssetFilter `json:"overrides"`
}

// GitHubSource represents a software package which is released via GitHub
// releases.
type GitHubSource struct {
	// Repo is the path to a repo in GitHub, using the "org/repo" format.
	Repo string `json:"repo"`
}

// S3Source represents a set of yaml documents published to a blob storage
// bucket.
type S3Source struct {
	// Bucket is the path to a Blob storage bucket
	Bucket string
	// Prefix is the path within a Blob storage bucket
	Prefix string
}

// AssetFilter provides an interface for selecting and managing assets within a
// release.
type AssetFilter struct {
	// IncludeArtifacts is a set of regep patterns to select which artifacts
	// from a release should be downloaded and included. If no IncludeArtifacts
	// are supplied, all files from the release will be included.
	IncludeArtifacts []string `json:"include"`
	// ExcludeArtifacts is a set of regexp patterns to remove artifacts which
	// would otherwise be selected by IncludeArtifacts.
	ExcludeArtifacts []string `json:"exclude,omitempty"`

	// Rename provides a mechanism to remap artifacts from one filename to
	// another.
	Rename map[string]string
}

// ReadConfig reads a set of Packages (as a map from package name to package configuration) from a yaml file a the selected path
func ReadConfig(path string) (retval map[string]*Package, err error) {
	retval = map[string]*Package{}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(bytes, &retval)
	if err != nil {
		return
	}
	for k, v := range retval {
		if v == nil {
			return nil, errors.New("Empty package for " + k)
		}
		v.Name = k
	}
	return
}

// String implements fmt.Stringer.
func (p *Package) String() string {
	return p.Name
}

// String implements fmt.Stringer.
func (s *Source) String() string {
	if s.GitHub != (GitHubSource{}) {
		return s.GitHub.Repo
	}
	if s.S3 != (S3Source{}) {
		return s.S3.Bucket + "/" + s.S3.Prefix
	}
	return "~~error~~"
}

// OrgRepo returns the GitHub org and repo associated with this Source.
//
// NOTE: this is totally a smell, because it's GitHub specific. The GitHub
// adapter outside package should probably handle this split if needed for the
// github library.
func (s *Source) OrgRepo() (string, string) {
	split := strings.SplitN(s.GitHub.Repo, "/", 2)
	return split[0], split[1]
}

// Accept provides a method which can be supplied to `FilterAssets` to handle
// `IncludeArtifacts` and `ExcludeArtifacts`.
func (af *AssetFilter) Accept(name string) string {
	if newName := af.Rename[name]; newName != "" {
		name = newName
	}

	// TODO: pre-compile the regexps
	for _, p := range af.ExcludeArtifacts {
		if ok, _ := regexp.MatchString(p, name); ok {
			return ""
		}
	}
	for _, p := range af.IncludeArtifacts {
		if ok, _ := regexp.MatchString(p, name); ok {
			return name
		}
	}

	// If no include patterns are supplied, accept all non-excluded strings
	if len(af.IncludeArtifacts) == 0 {
		return name
	}
	return ""
}

// Accept determines the best acceptance function for the given version, taking
// into account overrides.
func (s *Source) Accept(ver string) func(name string) string {
	best := s.AssetFilter
	ver = semver.Canonical(ver)
	for v, ie := range s.Overrides {
		if ver == v {
			best = ie
			break
		}
		if semver.MajorMinor(ver) == v {
			// Match same Major/Minor without patch, but there might be an exact override later.
			best = ie
		}
	}

	return best.Accept
}
