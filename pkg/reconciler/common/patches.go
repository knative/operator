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

package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"

	"knative.dev/operator/pkg/apis/operator/base"
)

const (
	strategicMergeDeleteDirective  = "delete"
	strategicMergeReplaceDirective = "replace"
)

type resourceIdentity struct {
	apiVersion string
	kind       string
	namespace  string
	name       string
}

// ResourcePatchStages carries resources removed by Apply to the post-readiness
// Delete stage. Delete removes them directly so root-level delete directives
// also work when status has no previous manifest for obsolete-resource cleanup.
type ResourcePatchStages struct {
	protectedDeletes []base.PatchTarget
	deletedResources mf.Manifest
}

// NewResourcePatchStages creates state shared by the Apply and Delete stages.
// protectedDeletes identifies component resources that later reconciliation
// stages require and therefore cannot be deleted.
func NewResourcePatchStages(protectedDeletes ...base.PatchTarget) *ResourcePatchStages {
	return &ResourcePatchStages{protectedDeletes: protectedDeletes}
}

// Apply applies user-provided patches to the final manifest and records
// resources selected by root-level delete directives.
func (stages *ResourcePatchStages) Apply(_ context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	deletedResources, err := applyResourcePatches(manifest, instance.GetSpec().GetPatches(), stages.protectedDeletes)
	if err != nil {
		instance.GetStatus().MarkInstallFailed(err.Error())
		return err
	}
	stages.deletedResources = deletedResources
	return nil
}

// Delete removes resources recorded by Apply from the cluster.
func (stages *ResourcePatchStages) Delete(_ context.Context, _ *mf.Manifest, instance base.KComponent) error {
	if err := stages.deletedResources.Delete(); err != nil {
		err = fmt.Errorf("delete resources removed by patches: %w", err)
		instance.GetStatus().MarkInstallFailed(err.Error())
		return err
	}
	return nil
}

// ApplyResourcePatches applies user-provided patches in declaration order.
// Root-level strategic merge directives can delete a matching resource or
// replace it while preserving its identity.
func ApplyResourcePatches(manifest *mf.Manifest, patches []base.ResourcePatch) error {
	_, err := applyResourcePatches(manifest, patches, nil)
	return err
}

func applyResourcePatches(manifest *mf.Manifest, patches []base.ResourcePatch, protectedDeletes []base.PatchTarget) (mf.Manifest, error) {
	deletedResources := manifest.Filter(mf.Nothing)
	for patchIndex, resourcePatch := range patches {
		deletedResource, err := applyPatchToManifest(manifest, resourcePatch, protectedDeletes)
		if err != nil {
			return deletedResources, fmt.Errorf("patch %d target %s: %w", patchIndex, describeTarget(resourcePatch.Target), err)
		}
		if deletedResource != nil {
			deletedResources = deletedResources.Append(*deletedResource)
		}
	}
	return deletedResources, nil
}

// applyPatchToManifest removes or rewrites the single resource selected by the
// patch target.
func applyPatchToManifest(manifest *mf.Manifest, resourcePatch base.ResourcePatch, protectedDeletes []base.PatchTarget) (*mf.Manifest, error) {
	matchesTarget := resourceMatchesTarget(resourcePatch.Target)
	matchedResources := manifest.Filter(matchesTarget).Resources()
	if len(matchedResources) == 0 {
		return nil, errors.New("did not match any generated resource")
	}
	if len(matchedResources) > 1 {
		if resourcePatch.Target.Namespace == "" {
			return nil, fmt.Errorf("matched %d generated resources; specify target.namespace", len(matchedResources))
		}
		return nil, fmt.Errorf("matched %d generated resources; target must match exactly one resource", len(matchedResources))
	}

	patchJSON, err := yaml.YAMLToJSON([]byte(resourcePatch.Patch.Content))
	if err != nil {
		return nil, fmt.Errorf("convert patch content to JSON: %w", err)
	}
	if err := validatePatchDirectives(resourcePatch.Patch.Type, patchJSON); err != nil {
		return nil, err
	}
	rootDirective, patchDocument, err := parseRootPatchDirective(resourcePatch.Patch.Type, patchJSON)
	if err != nil {
		return nil, err
	}

	if rootDirective == strategicMergeDeleteDirective {
		if mf.CRDs(&matchedResources[0]) {
			return nil, errors.New("deleting CustomResourceDefinitions is not supported")
		}
		if matchedResources[0].GroupVersionKind() == corev1.SchemeGroupVersion.WithKind("Namespace") {
			return nil, errors.New("deleting Namespaces is not supported")
		}
		for _, protectedTarget := range protectedDeletes {
			if resourceMatchesTarget(protectedTarget)(&matchedResources[0]) {
				return nil, fmt.Errorf("deleting protected resource %s is not supported", describeTarget(protectedTarget))
			}
		}
		deletedResource := manifest.Filter(matchesTarget)
		*manifest = manifest.Filter(mf.Not(matchesTarget))
		return &deletedResource, nil
	}
	if rootDirective == strategicMergeReplaceDirective {
		patchJSON, err = preserveResourceIdentityInReplacement(patchDocument, &matchedResources[0])
		if err != nil {
			return nil, err
		}
	}

	patchedManifest, err := manifest.Transform(func(resource *unstructured.Unstructured) error {
		if !matchesTarget(resource) {
			return nil
		}
		return applyPatchToResource(resource, resourcePatch.Patch.Type, patchJSON)
	})
	if err != nil {
		return nil, err
	}
	*manifest = patchedManifest
	return nil, nil
}

// resourceMatchesTarget returns a predicate for selecting the generated
// resource a patch target refers to. An empty target namespace matches any
// namespace.
func resourceMatchesTarget(target base.PatchTarget) mf.Predicate {
	return func(resource *unstructured.Unstructured) bool {
		if resource.GetAPIVersion() != target.APIVersion {
			return false
		}
		if resource.GetKind() != target.Kind {
			return false
		}
		if resource.GetName() != target.Name {
			return false
		}
		return target.Namespace == "" || resource.GetNamespace() == target.Namespace
	}
}

func applyPatchToResource(resource *unstructured.Unstructured, patchType base.PatchType, patchJSON []byte) error {
	originalIdentity := identityOf(resource)
	originalJSON, err := resource.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal generated resource: %w", err)
	}

	var patchedJSON []byte
	switch patchType {
	case base.JSONPatchType:
		decoded, err := jsonpatch.DecodePatch(patchJSON)
		if err != nil {
			return fmt.Errorf("decode JSON patch: %w", err)
		}
		patchedJSON, err = decoded.Apply(originalJSON)
		if err != nil {
			return fmt.Errorf("apply JSON patch: %w", err)
		}
	case base.MergePatchType:
		patchedJSON, err = jsonpatch.MergePatch(originalJSON, patchJSON)
		if err != nil {
			return fmt.Errorf("apply merge patch: %w", err)
		}
	case base.StrategicMergePatchType:
		resourceType, err := scheme.Scheme.New(resource.GroupVersionKind())
		if runtime.IsNotRegisteredError(err) {
			return fmt.Errorf(
				"strategic merge patch is not supported for %s; use a json or merge patch",
				resource.GroupVersionKind(),
			)
		}
		if err != nil {
			return fmt.Errorf("resolve strategic merge patch schema for %s: %w", resource.GroupVersionKind(), err)
		}
		patchedJSON, err = strategicpatch.StrategicMergePatch(originalJSON, patchJSON, resourceType)
		if err != nil {
			return fmt.Errorf("apply strategic merge patch: %w", err)
		}
	default:
		return fmt.Errorf("unsupported patch type %q", patchType)
	}

	patched := &unstructured.Unstructured{}
	if err := patched.UnmarshalJSON(patchedJSON); err != nil {
		return fmt.Errorf("decode patched resource: %w", err)
	}
	if patchedIdentity := identityOf(patched); patchedIdentity != originalIdentity {
		return errors.New("patch must not change apiVersion, kind, metadata.name, or metadata.namespace")
	}
	*resource = *patched
	return nil
}

// validatePatchDirectives prevents strategic merge directives from becoming
// literal resource fields when another patch algorithm is selected.
func validatePatchDirectives(patchType base.PatchType, patchJSON []byte) error {
	if patchType != base.StrategicMergePatchType {
		var patchDocument interface{}
		if err := json.Unmarshal(patchJSON, &patchDocument); err != nil {
			return fmt.Errorf("decode patch document: %w", err)
		}
		if containsPatchDirective(patchDocument) {
			return errors.New("$patch directive is only supported for the strategic merge patch type")
		}
	}
	return nil
}

func parseRootPatchDirective(patchType base.PatchType, patchJSON []byte) (string, map[string]interface{}, error) {
	if patchType != base.StrategicMergePatchType {
		return "", nil, nil
	}
	patchDocument := map[string]interface{}{}
	if err := json.Unmarshal(patchJSON, &patchDocument); err != nil {
		return "", nil, fmt.Errorf("decode strategic merge patch: %w", err)
	}
	directive, found := patchDocument["$patch"]
	if !found {
		return "", patchDocument, nil
	}
	directiveString, ok := directive.(string)
	if !ok {
		return "", nil, errors.New("root $patch directive must be a string")
	}
	return directiveString, patchDocument, nil
}

func containsPatchDirective(value interface{}) bool {
	switch value := value.(type) {
	case map[string]interface{}:
		for key, nestedValue := range value {
			if key == "$patch" || containsPatchDirective(nestedValue) {
				return true
			}
		}
	case []interface{}:
		for _, nestedValue := range value {
			if containsPatchDirective(nestedValue) {
				return true
			}
		}
	}
	return false
}

// preserveResourceIdentityInReplacement lets a root-level replacement omit
// identity fields already specified by its target. Explicit identity fields
// remain untouched so applyPatchToResource can reject attempts to change them.
func preserveResourceIdentityInReplacement(patchDocument map[string]interface{}, resource *unstructured.Unstructured) ([]byte, error) {
	identity := identityOf(resource)
	if _, found := patchDocument["apiVersion"]; !found {
		patchDocument["apiVersion"] = identity.apiVersion
	}
	if _, found := patchDocument["kind"]; !found {
		patchDocument["kind"] = identity.kind
	}

	metadataValue, found := patchDocument["metadata"]
	if !found {
		metadataValue = map[string]interface{}{}
		patchDocument["metadata"] = metadataValue
	}
	metadata, ok := metadataValue.(map[string]interface{})
	if !ok {
		return nil, errors.New("root $patch: replace metadata must be an object")
	}
	if _, found := metadata["name"]; !found {
		metadata["name"] = identity.name
	}
	if identity.namespace != "" {
		if _, found := metadata["namespace"]; !found {
			metadata["namespace"] = identity.namespace
		}
	}

	patchJSON, err := json.Marshal(patchDocument)
	if err != nil {
		return nil, fmt.Errorf("encode root replacement patch: %w", err)
	}
	return patchJSON, nil
}

func identityOf(resource *unstructured.Unstructured) resourceIdentity {
	return resourceIdentity{
		apiVersion: resource.GetAPIVersion(),
		kind:       resource.GetKind(),
		namespace:  resource.GetNamespace(),
		name:       resource.GetName(),
	}
}

func describeTarget(target base.PatchTarget) string {
	namespace := target.Namespace
	if namespace == "" {
		namespace = "*"
	}
	return fmt.Sprintf("%s, Kind=%s, %s/%s", target.APIVersion, target.Kind, namespace, target.Name)
}
