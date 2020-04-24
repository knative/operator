// +build tools

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

package tools

// This package imports things required by this repository, to force `go mod` to see them as dependencies
import (
	_ "k8s.io/code-generator"
	_ "knative.dev/test-infra/scripts"
	_ "knative.dev/pkg/codegen/cmd/injection-gen"
	_ "knative.dev/caching/pkg/apis/caching/v1alpha1"
	_ "knative.dev/test-infra/tools/dep-collector"
	_ "k8s.io/apimachinery/pkg/util/sets/types"
	_ "knative.dev/pkg/hack"
)
