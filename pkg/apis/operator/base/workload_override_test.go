/*
Copyright 2026 The Knative Authors

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

import (
	"testing"

	"sigs.k8s.io/yaml"
)

// TestWorkloadOverride_GhostFieldsUnmarshal verifies that the deprecated
// no-op fields `version` and `volumeMounts` on WorkloadOverride continue to
// unmarshal without error. Existing KnativeServing / KnativeEventing CRs in
// the wild still set these fields; the CRD migration to controller-gen must
// not reject them.
func TestWorkloadOverride_GhostFieldsUnmarshal(t *testing.T) {
	raw := []byte(`
name: controller
version: "1.16.0"
volumeMounts:
- name: config
  mountPath: /etc/config
`)

	var wo WorkloadOverride
	if err := yaml.Unmarshal(raw, &wo); err != nil {
		t.Fatalf("failed to unmarshal legacy WorkloadOverride yaml: %v", err)
	}

	if wo.Name != "controller" {
		t.Errorf("Name = %q, want controller", wo.Name)
	}
	if wo.Version != "1.16.0" {
		t.Errorf("Version = %q, want 1.16.0", wo.Version)
	}
	if len(wo.VolumeMounts) != 1 {
		t.Fatalf("len(VolumeMounts) = %d, want 1", len(wo.VolumeMounts))
	}
	vm := wo.VolumeMounts[0]
	if vm.Name != "config" {
		t.Errorf("VolumeMounts[0].Name = %q, want config", vm.Name)
	}
	if vm.MountPath != "/etc/config" {
		t.Errorf("VolumeMounts[0].MountPath = %q, want /etc/config", vm.MountPath)
	}
}

// TestWorkloadOverride_GhostFieldsStrictUnmarshal reproduces what
// sigs.k8s.io/yaml does for CRD-based admission: the strict path must also
// accept the deprecated fields, not just drop them silently.
func TestWorkloadOverride_GhostFieldsStrictUnmarshal(t *testing.T) {
	raw := []byte(`
name: controller
version: "1.16.0"
volumeMounts:
- name: config
  mountPath: /etc/config
`)

	var wo WorkloadOverride
	if err := yaml.UnmarshalStrict(raw, &wo); err != nil {
		t.Fatalf("strict unmarshal failed: %v", err)
	}
	if wo.Version == "" || len(wo.VolumeMounts) == 0 {
		t.Fatalf("ghost fields lost on strict unmarshal: %+v", wo)
	}
}
