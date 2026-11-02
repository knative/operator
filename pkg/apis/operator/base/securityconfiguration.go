/*
Copyright 2022 The Knative Authors

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

package base

// SecurityGuardConfiguration specifies options for the security guard component.
type SecurityGuardConfiguration struct {
	// Enabled is retained for backward compatibility and has no effect.
	// Deprecated: Security Guard is no longer supported as of Knative Operator version 1.25.
	// This field has no effect regardless of the target Knative Serving version.
	Enabled bool `json:"enabled"`
}
