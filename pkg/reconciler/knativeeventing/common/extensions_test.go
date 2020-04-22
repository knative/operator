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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
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

	ke := &v1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-eventing",
			Namespace: "knative-eventing",
		},
	}

	logger := logging.FromContext(ctx).
		Named("test-controller").
		With(zap.String(logkey.ControllerType, "test-controller"))

	results, err := platform.Transformers(kubeclient.Get(ctx), ke, logger)
	util.AssertEqual(t, err, nil)
	// By default, there are 5 functions.
	util.AssertEqual(t, len(results), 5)

	platform = append(platform, fakePlatform)
	results, err = platform.Transformers(kubeclient.Get(ctx), ke, logger)
	util.AssertEqual(t, err, nil)
	// There is one function in existing platform, so there will be 6 functions in total.
	util.AssertEqual(t, len(results), 6)

	platformErr = append(platformErr, fakePlatformErr)
	results, err = platformErr.Transformers(kubeclient.Get(ctx), ke, logger)
	util.AssertEqual(t, err.Error(), "Test Error")
	// By default, there are 5 functions.
	util.AssertEqual(t, len(results), 5)
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
