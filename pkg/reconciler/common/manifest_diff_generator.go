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
	"context"
	"fmt"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"

	"github.com/go-logr/zapr"
	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ManifestDiffGenerator generates a list of resources to create, a list of resource to patch, and
// a list of resources to delete.
func ManifestDiffGenerator(oldManifest, newManifest mf.Manifest) ([]unstructured.Unstructured,
	[]unstructured.Unstructured, []unstructured.Unstructured) {
	listResourceCreate := []unstructured.Unstructured{}
	listResourcePatch := []unstructured.Unstructured{}
	listResourceDelete := []unstructured.Unstructured{}

	oldResources := oldManifest.Resources()
	newResources := newManifest.Resources()

	for _, resource := range newResources {
		if found, _ := FindResourceByNSGroupKindName(resource, oldResources); !found {
			listResourceCreate = append(listResourceDelete, resource)
		} else {
			listResourcePatch = append(listResourcePatch, resource)
		}
	}

	for _, resource := range oldResources {
		if found, _ := FindResourceByNSGroupKindName(resource, newResources); !found {
			listResourceDelete = append(listResourceDelete, resource)
		}
	}

	return listResourceCreate, listResourcePatch, listResourceDelete
}

// FindResourceByNSGroupKindName is a lookup function, which returns true and the old resource, if the resource is available in an array of
// unstructured.Unstructured, matching by APIVersion, Kind and Name.
func FindResourceByNSGroupKindName(r unstructured.Unstructured, manifestResource []unstructured.Unstructured) (bool, unstructured.Unstructured) {
	mapMenifestResource := make(map[string]unstructured.Unstructured, len(manifestResource))
	for _, resource := range manifestResource {
		key := fmt.Sprintf("%s%s%s", resource.GetAPIVersion(),
			resource.GetKind(), resource.GetName())
		mapMenifestResource[key] = resource
	}
	key := fmt.Sprintf("%s%s%s", r.GetAPIVersion(), r.GetKind(), r.GetName())
	if val, found := mapMenifestResource[key]; found {
		return true, val
	}
	return false, unstructured.Unstructured{}
}

func CreateManifestByVersion(config *rest.Config, ctx context.Context, version string) (mf.Manifest, error) {
	logger := logging.FromContext(ctx)
	koDataDir := os.Getenv("KO_DATA_PATH")
	manifest, err := mfc.NewManifest(filepath.Join(koDataDir, "knative-serving/"+"v"+version),
		config,
		mf.UseLogger(zapr.NewLogger(logger.Desugar()).WithName("manifestival")))
	if err != nil {
		logger.Fatalw("Error creating the Manifest for knative-serving", zap.Error(err))
		return mf.Manifest{}, err
	}
	return manifest, err
}
