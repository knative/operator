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
	"errors"
	"testing"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	util "knative.dev/operator/pkg/reconciler/common/testing"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	_ "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/logging/logkey"
)

var (
	platform    Platforms
	platformErr Platforms
)

func TestTransformers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &rest.Config{}
	ctx, _ = injection.Fake.SetupInformers(ctx, cfg)

	logger := logging.FromContext(ctx).
		Named("test-controller").
		With(zap.String(logkey.ControllerType, "test-controller"))

	results, err := platform.Transformers(kubeclient.Get(ctx), logger)
	util.AssertEqual(t, err, nil)
	util.AssertEqual(t, len(results), 0)

	platform = append(platform, fakePlatform)
	results, err = platform.Transformers(kubeclient.Get(ctx), logger)
	util.AssertEqual(t, err, nil)
	util.AssertEqual(t, len(results), 1)

	platformErr = append(platformErr, fakePlatformErr)
	results, err = platformErr.Transformers(kubeclient.Get(ctx), logger)
	util.AssertEqual(t, err.Error(), "Test Error")
	util.AssertEqual(t, len(results), 0)
}

func fakePlatformErr(kubeClient kubernetes.Interface, logger *zap.SugaredLogger) (mf.Transformer, error) {
	return fakeTransformer(), errors.New("Test Error")
}

func fakePlatform(kubeClient kubernetes.Interface, logger *zap.SugaredLogger) (mf.Transformer, error) {
	return fakeTransformer(), nil
}

func fakeTransformer() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		return nil
	}
}
