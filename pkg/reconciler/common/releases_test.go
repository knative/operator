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
	"os"
	"testing"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

const (
	SERVING_CORE         = "testdata/kodata/knative-serving/0.16.1/serving-core.yaml"
	SERVING_HPA          = "testdata/kodata/knative-serving/0.16.1/serving-hpa.yaml"
	EVENTING_CORE        = "testdata/kodata/knative-eventing/0.16.0/eventing-core.yaml"
	IN_MEMORY_CHANNEL    = "testdata/kodata/knative-eventing/0.16.0/in-memory-channel.yaml"
	SERVING_VERSION_CORE = "testdata/kodata/knative-serving/" + VersionVariable + "/serving-core.yaml"
	SERVING_VERSION_HPA  = "testdata/kodata/knative-serving/" + VersionVariable + "/serving-hpa.yaml"
)

func TestRetrieveManifestPath(t *testing.T) {
	koPath := "testdata/kodata"
	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)

	tests := []struct {
		component v1alpha1.KComponent
		version   string
		name      string
		expected  string
	}{{
		name:      "Valid Knative Serving Version",
		component: &v1alpha1.KnativeServing{},
		version:   "0.14.0",
		expected:  koPath + "/knative-serving/0.14.0",
	}, {
		name:      "Valid Knative Eventing Version",
		component: &v1alpha1.KnativeEventing{},
		version:   "0.14.2",
		expected:  koPath + "/knative-eventing/0.14.2",
	}, {
		name: "Valid Knative Serving URLs",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Manifests: []v1alpha1.Manifest{{
						Url: SERVING_CORE,
					}, {
						Url: SERVING_HPA,
					}},
				},
			},
		},
		version:  "0.16.0",
		expected: SERVING_CORE + "," + SERVING_HPA,
	}, {
		name: "Valid Knative Eventing URLs",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Manifests: []v1alpha1.Manifest{{
						Url: EVENTING_CORE,
					}, {
						Url: IN_MEMORY_CHANNEL,
					}},
				},
			},
		},
		version:  "0.16.0",
		expected: EVENTING_CORE + "," + IN_MEMORY_CHANNEL,
	}, {
		name: "Valid Knative Serving URLs with the version parameter",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Manifests: []v1alpha1.Manifest{{
						Url: SERVING_VERSION_CORE,
					}, {
						Url: SERVING_VERSION_HPA,
					}},
				},
			},
		},
		version:  "0.16.1",
		expected: SERVING_CORE + "," + SERVING_HPA,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifestPath := targetManifestPath(test.version, test.component)
			util.AssertEqual(t, manifestPath, test.expected)
			manifest, err := mf.NewManifest(manifestPath)
			util.AssertEqual(t, err, nil)
			util.AssertEqual(t, len(manifest.Resources()) > 0, true)
		})
	}

	invalidPathTests := []struct {
		component v1alpha1.KComponent
		version   string
		name      string
		expected  string
	}{{
		name:      "Invalid Knative Serving Version",
		component: &v1alpha1.KnativeServing{},
		version:   "invalid-version",
		expected:  "",
	}}

	for _, test := range invalidPathTests {
		t.Run(test.name, func(t *testing.T) {
			manifestPath := targetManifestPath(test.version, test.component)
			util.AssertEqual(t, manifestPath, test.expected)
			manifest, err := mf.NewManifest(manifestPath)
			util.AssertEqual(t, err != nil, true)
			util.AssertEqual(t, len(manifest.Resources()) == 0, true)
		})
	}
}

func TestTargetVersion(t *testing.T) {
	koPath := "testdata/kodata"

	tests := []struct {
		name      string
		component v1alpha1.KComponent
		expected  string
	}{{
		name: "serving",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.16",
				},
			},
		},
		expected: "0.16.1",
	}, {
		name: "eventing",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.15",
				},
			},
		},
		expected: "0.15.0",
	}, {
		name:      "eventing",
		component: &v1alpha1.KnativeEventing{},
		expected:  "0.16.0",
	}, {
		name: "serving",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.16",
					Manifests: []v1alpha1.Manifest{{
						Url: SERVING_VERSION_CORE,
					}, {
						Url: SERVING_VERSION_HPA,
					}},
				},
			},
		},
		expected: "0.16",
	}, {
		name: "serving",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "",
					Manifests: []v1alpha1.Manifest{{
						Url: SERVING_VERSION_CORE,
					}, {
						Url: SERVING_VERSION_HPA,
					}},
				},
			},
		},
		expected: "",
	}, {
		name: "serving CR with major.minor version not available",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.12",
				},
			},
		},
		expected: "0.12",
	}, {
		name: "serving CR with the version latest",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "latest",
				},
			},
		},
		expected: "latest",
	}, {
		name: "eventing CR with major.minor version not available",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.12",
				},
			},
		},
		expected: "0.12",
	}, {
		name: "serving CR with major.minor.patch version not available",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.12.0",
				},
			},
		},
		expected: "0.12.0",
	}, {
		name: "eventing CR with major.minor.patch version not available",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.12.1",
				},
			},
		},
		expected: "0.12.1",
	}, {
		name: "eventing CR with the version latest",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "latest",
				},
			},
		},
		expected: "latest",
	}}

	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version := TargetVersion(test.component)
			util.AssertEqual(t, version, test.expected)
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	koPath := "testdata/kodata"

	tests := []struct {
		name      string
		component v1alpha1.KComponent
		expected  string
	}{{
		name: "serving",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.16",
				},
			},
		},
		expected: "0.16.1",
	}, {
		name: "eventing",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.15",
				},
			},
		},
		expected: "0.15.0",
	}, {
		name: "eventing CR with the major.minor version not available",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.13",
				},
			},
		},
		expected: "0.13",
	}, {
		name: "serving CR with the major.minor version not available",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.13",
				},
			},
		},
		expected: "0.13",
	}, {
		name: "eventing CR with the major.minor.patch version not available",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.13.1",
				},
			},
		},
		expected: "0.13.1",
	}, {
		name: "serving CR with the major.minor.patch version not available",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.13.1",
				},
			},
		},
		expected: "0.13.1",
	}}

	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version := getLatestRelease(test.component, test.component.GetSpec().GetVersion())
			util.AssertEqual(t, version, test.expected)
		})
	}
}

func TestLatestRelease(t *testing.T) {
	koPath := "testdata/kodata"

	tests := []struct {
		name      string
		component v1alpha1.KComponent
		expected  string
	}{{
		name:      "serving",
		component: &v1alpha1.KnativeServing{},
		expected:  "0.16.1",
	}, {
		name:      "eventing",
		component: &v1alpha1.KnativeEventing{},
		expected:  "0.16.0",
	}}

	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version := latestRelease(test.component)
			util.AssertEqual(t, version, test.expected)
		})
	}
}

func TestListReleases(t *testing.T) {
	koPath := "testdata/kodata"

	tests := []struct {
		name      string
		component v1alpha1.KComponent
		expected  []string
	}{{
		name:      "knative-serving",
		component: &v1alpha1.KnativeServing{},
		expected:  []string{"0.16.1", "0.16.0", "0.15.0", "0.14.0", "latest"},
	}, {
		name:      "knative-eventing",
		component: &v1alpha1.KnativeEventing{},
		expected:  []string{"0.16.0", "0.15.0", "0.14.2", "latest"},
	}}

	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version, err := allReleases(test.component)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, version, test.expected)
		})
	}
}

func TestIsVersionValidMigrationEligible(t *testing.T) {
	koPath := "testdata/kodata"
	tests := []struct {
		name      string
		component v1alpha1.KComponent
		expected  bool
	}{{
		name: "knative-serving with target version in major.minor.patch and without status.version",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.14.2",
				},
			},
		},
		expected: true,
	}, {
		name: "knative-serving upgrading one minor version",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.14.2",
				},
			},
			Status: v1alpha1.KnativeServingStatus{
				Version: "0.13.0",
			},
		},
		expected: true,
	}, {
		name: "knative-serving upgrading across multiple minor versions",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.15.0",
				},
			},
			Status: v1alpha1.KnativeServingStatus{
				Version: "0.13.0",
			},
		},
		expected: false,
	}, {
		name: "knative-serving with the version latest upgrading to",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "latest",
				},
			},
			Status: v1alpha1.KnativeServingStatus{
				Version: "0.13.0",
			},
		},
		expected: true,
	}, {
		name: "knative-serving with the version latest upgrading from",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.14.0",
				},
			},
			Status: v1alpha1.KnativeServingStatus{
				Version: "latest",
			},
		},
		expected: true,
	}, {
		name: "knative-serving with the version latest",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "latest",
				},
			},
		},
		expected: true,
	}, {
		name: "knative-serving upgrading to the latest version across multiple minor versions",
		component: &v1alpha1.KnativeServing{
			Status: v1alpha1.KnativeServingStatus{
				Version: "0.13.0",
			},
		},
		// The latest version is 0.15.0
		expected: false,
	}, {
		name: "knative-serving upgrading to the latest version",
		component: &v1alpha1.KnativeServing{
			Status: v1alpha1.KnativeServingStatus{
				Version: "0.15.0",
			},
		},
		// The latest version is 0.16.0
		expected: true,
	}, {
		name:      "knative-serving with latest version and empty status.version",
		component: &v1alpha1.KnativeServing{},
		expected:  true,
	}, {
		name: "knative-serving with the same status.version and spec.version",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.15.0",
				},
			},
			Status: v1alpha1.KnativeServingStatus{
				Version: "0.15.0",
			},
		},
		expected: true,
	}, {
		name: "knative-serving with target version in major.minor",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.15",
				},
			},
		},
		expected: true,
	}, {
		name: "knative-serving with invalid target version",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "badVersion",
				},
			},
		},
		expected: false,
	}, {
		name: "knative-serving with invalid target version only major",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "1",
				},
			},
		},
		expected: false,
	}}

	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsVersionValidMigrationEligible(test.component)
			util.AssertEqual(t, result == nil, test.expected)
		})
	}
}

func TestTargetManifest(t *testing.T) {
	tests := []struct {
		name                 string
		component            v1alpha1.KComponent
		expectedNumResources int
		expectedError        error
	}{{
		name: "knative-serving with spec.manifests matched",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.16.0",
					Manifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/knative-serving/" + VersionVariable + "/serving-core.yaml",
					}, {
						Url: "testdata/kodata/knative-serving/" + VersionVariable + "/serving-hpa.yaml",
					}},
				},
			},
		},
		expectedNumResources: 2,
		expectedError:        nil,
	}, {
		name: "knative-serving with spec.manifests unmatched",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.16.0",
					Manifests: []v1alpha1.Manifest{{
						Url: "testdata/invalid_kodata/knative-serving/" + VersionVariable + "_unmatched_version/serving-core.yaml",
					}, {
						Url: "testdata/invalid_kodata/knative-serving/" + VersionVariable + "_unmatched_version/serving-hpa.yaml",
					}},
				},
			},
		},
		expectedNumResources: 0,
		expectedError: fmt.Errorf("The version of the manifests %s does not match the target "+
			"version of the operator CR %s. The resource name is %s.", "v0.17.2", "v0.16.0", "knative-serving"),
	}, {
		name: "knative-serving with spec.manifests matched but no spec.version",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Manifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/knative-serving/0.16.0/serving-core.yaml",
					}, {
						Url: "testdata/kodata/knative-serving/0.16.0/serving-hpa.yaml",
					}},
				},
			},
		},
		expectedNumResources: 2,
		expectedError:        nil,
	}, {
		name: "knative-serving with additional resources",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Manifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/knative-serving/0.16.1/serving-core.yaml",
					}, {
						Url: "testdata/kodata/knative-serving/0.16.1/serving-hpa.yaml",
					}, {
						Url: "testdata/kodata/knative-serving/0.16.1/serving-crd.yaml",
					}},
				},
			},
		},
		expectedNumResources: 3,
		expectedError:        nil,
	}, {
		name: "knative-serving with spec.version available",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.16.0",
				},
			},
		},
		expectedNumResources: 2,
		expectedError:        nil,
	}, {
		name: "knative-serving with major.minor spec.version not available",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.12",
				},
			},
		},
		expectedNumResources: 0,
		expectedError: fmt.Errorf("The manifests of the target version %v are not available to this release.",
			"0.12"),
	}, {
		name: "knative-serving with major.minor.patch spec.version not available",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.12.1",
				},
			},
		},
		expectedNumResources: 0,
		expectedError: fmt.Errorf("The manifests of the target version %v are not available to this release.",
			"0.12.1"),
	}, {
		name: "knative-serving with the latest version available",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "latest",
				},
			},
		},
		expectedNumResources: 2,
		expectedError:        nil,
	}, {
		name: "knative-eventing with major.minor spec.version not available",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.12",
				},
			},
		},
		expectedNumResources: 0,
		expectedError: fmt.Errorf("The manifests of the target version %v are not available to this release.",
			"0.12"),
	}, {
		name: "knative-eventing with major.minor.patch spec.version not available",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.12.1",
				},
			},
		},
		expectedNumResources: 0,
		expectedError: fmt.Errorf("The manifests of the target version %v are not available to this release.",
			"0.12.1"),
	}, {
		name: "knative-eventing with the latest available",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "latest",
				},
			},
		},
		expectedNumResources: 2,
		expectedError:        nil,
	}, {
		name: "knative-serving with additional manifests only",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					AdditionalManifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedNumResources: 3,
		expectedError:        nil,
	}, {
		name: "knative-serving with manifests and additional manifests",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Manifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/knative-serving/0.16.1/serving-core.yaml",
					}},
					AdditionalManifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedNumResources: 1,
		expectedError:        nil,
	}, {
		name: "knative-eventing with additional manifests only",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					AdditionalManifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedNumResources: 2,
		expectedError:        nil,
	}, {
		name: "knative-eventing with manifests and additional manifests",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Manifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/knative-eventing/0.16.0/eventing-core.yaml",
					}},
					AdditionalManifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedNumResources: 1,
		expectedError:        nil,
	}}

	koPath := "testdata/kodata"
	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m, err := TargetManifest(test.component)
			util.AssertEqual(t, len(m.Resources()), test.expectedNumResources)
			if err != test.expectedError {
				if err != nil {
					util.AssertEqual(t, err.Error(), test.expectedError.Error())
				} else {
					util.AssertEqual(t, nil, test.expectedError.Error())
				}
			}
		})
	}
}

func TestTargetAdditionalManifest(t *testing.T) {
	tests := []struct {
		name                 string
		component            v1alpha1.KComponent
		expectedNumResources int
		expectedError        error
	}{{
		name: "knative-serving with additional manifests only",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					AdditionalManifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedNumResources: 1,
		expectedError:        nil,
	}, {
		name: "knative-serving with manifests and additional manifests",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Manifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/knative-serving/0.16.1/serving-core.yaml",
					}},
					AdditionalManifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedNumResources: 1,
		expectedError:        nil,
	}, {
		name: "knative-eventing with additional manifests only",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					AdditionalManifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedNumResources: 1,
		expectedError:        nil,
	}, {
		name: "knative-eventing with manifests and additional manifests",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Manifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/knative-eventing/0.16.0/eventing-core.yaml",
					}},
					AdditionalManifests: []v1alpha1.Manifest{{
						Url: "testdata/kodata/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedNumResources: 1,
		expectedError:        nil,
	}, {
		name: "knative-serving with the latest version available",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "latest",
				},
			},
		},
		expectedNumResources: 0,
		expectedError:        nil,
	}}

	koPath := "testdata/kodata"
	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m, err := TargetAdditionalManifest(test.component)
			util.AssertEqual(t, len(m.Resources()), test.expectedNumResources)
			if err != test.expectedError {
				if err != nil {
					util.AssertEqual(t, err.Error(), test.expectedError.Error())
				} else {
					util.AssertEqual(t, nil, test.expectedError.Error())
				}
			}
		})
	}
}
