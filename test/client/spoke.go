//go:build e2e && multicluster
// +build e2e,multicluster

/*
Copyright 2025 The Knative Authors

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

package client

import (
	"os"
	"testing"

	"knative.dev/operator/test"
)

func SetupSpoke(t *testing.T) *test.Clients {
	t.Helper()
	path := os.Getenv("SPOKE_HOST_KUBECONFIG")
	if path == "" {
		t.Fatalf("SPOKE_HOST_KUBECONFIG must be set")
	}
	clients, err := test.NewClients(path, "")
	if err != nil {
		t.Fatalf("Couldn't initialize spoke clients: %v", err)
	}
	return clients
}
