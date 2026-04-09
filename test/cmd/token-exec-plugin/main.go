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

// token-exec-plugin reads a bearer token from a file and writes an
// ExecCredential JSON to stdout.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <token-file-path>\n", os.Args[0])
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read token %q: %v\n", os.Args[1], err)
		os.Exit(1)
	}
	cred := map[string]any{
		"apiVersion": "client.authentication.k8s.io/v1",
		"kind":       "ExecCredential",
		"status": map[string]any{
			"token": strings.TrimSpace(string(raw)),
		},
	}
	if err := json.NewEncoder(os.Stdout).Encode(cred); err != nil {
		fmt.Fprintf(os.Stderr, "encode credential: %v\n", err)
		os.Exit(1)
	}
}
