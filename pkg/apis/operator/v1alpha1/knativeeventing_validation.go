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

package v1alpha1

import (
	"context"
	"knative.dev/pkg/apis"
)

// Validate makes sure that KnativeEventing is properly configured.
func (c *KnativeEventing) Validate(ctx context.Context) (errs *apis.FieldError) {
	if apis.IsInCreate(ctx) {
		// TODO: when the CR is created, implement the code here
	}

	if apis.IsInUpdate(ctx) {
		// TODO: when the CR is updated, implement the code here
	}
	if apis.IsInDelete(ctx) {
		// TODO: when the CR is deleted, implement the code here
	}

	return errs
}

// Validate implements apis.Validatable
func (cs *KnativeEventingStatus) Validate(ctx context.Context) *apis.FieldError {
	// TODO: leave it empty for future implementation
	return nil
}
