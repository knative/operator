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
	"os"
	"path/filepath"

	"github.com/go-logr/zapr"
	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"

	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

// RetrieveManifest returns the manifest for Knative based a provided version and component
func RetrieveManifest(ctx context.Context, version, kcomponent string) (mf.Manifest, error) {
	logger := logging.FromContext(ctx)
	koDataDir := os.Getenv("KO_DATA_PATH")
	manifest, err := mfc.NewManifest(filepath.Join(koDataDir, kcomponent, version),
		injection.GetConfig(ctx),
		mf.UseLogger(zapr.NewLogger(logger.Desugar()).WithName("manifestival")))

	if err != nil {
		return manifest, err
	}

	if len(manifest.Resources()) == 0 {
		return manifest, fmt.Errorf("unable to find the manifest for the Knative version %s", version)
	}

	return manifest, nil
}
