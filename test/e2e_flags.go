/*
Copyright 2019 The Knative Authors

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

// This file contains logic to encapsulate flags which are needed to specify
// what cluster, etc. to use for e2e tests.

package test

import (
	"flag"
	"os"
)

var (
	// ServingOperatorNamespace is the default namespace for serving operator e2e tests
	ServingOperatorNamespace  = getenv("TEST_NAMESPACE", "knative-operator-testing")
	EventingOperatorNamespace = getenv("TEST_EVENTING_NAMESPACE", ServingOperatorNamespace)
	// OperatorName is the default operator name for serving operator e2e tests
	OperatorName = getenv("TEST_RESOURCE", "knative")
	// OperatorFlags holds the flags or defaults for knative/operator settings in the user's environment.
	OperatorFlags = initializeOperatorFlags()
)

func getenv(name, defaultValue string) string {
	value, set := os.LookupEnv(name)
	if !set {
		value = defaultValue
	}
	return value
}

// OperatorEnvironmentFlags holds the e2e flags needed only by the operator repo.
type OperatorEnvironmentFlags struct {
	PreviousServingVersion  string // Indicates the previous version of Knative Serving.
	PreviousEventingVersion string // Indicates the previous version of Knative Eventing.
}

func initializeOperatorFlags() *OperatorEnvironmentFlags {
	var f OperatorEnvironmentFlags

	flag.StringVar(&f.PreviousServingVersion, "preservingversion", "",
		"Set this flag to the previous version of Knative Serving.")
	flag.StringVar(&f.PreviousEventingVersion, "preeventingversion", "",
		"Set this flag to the previous version of Knative Eventing.")

	return &f
}
