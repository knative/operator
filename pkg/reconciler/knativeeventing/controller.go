/*
Copyright 2019 The Knative Authors.
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

package knativeeventing

import (
	"context"

	"github.com/go-logr/zapr"
	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/cache"

	"knative.dev/operator/pkg/apis/operator/v1beta1"
	operatorclient "knative.dev/operator/pkg/client/injection/client"
	knativeEventinginformer "knative.dev/operator/pkg/client/injection/informers/operator/v1beta1/knativeeventing"
	knereconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1beta1/knativeeventing"
	"knative.dev/operator/pkg/reconciler/common"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	deploymentinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment/filtered"
	configmapinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/configmap/filtered"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

const (
	// SelectorKey is the key of the selector for the KnativeEventing resources.
	SelectorKey = "app.kubernetes.io/name"
	// SelectorValue is the value of the selector for the KnativeEventing resources.
	SelectorValue = "knative-eventing"
	// Selector is the selector for the KnativeEventing resources.
	Selector = SelectorKey + "=" + SelectorValue
)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return NewExtendedControlle1r(common.NoExtension)(ctx, cmw)
}

// NewExtendedControlle1r returns a controller extended to a specific platform
func NewExtendedControlle1r(generator common.ExtensionGenerator) injection.ControllerConstructor {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		knativeEventingInformer := knativeEventinginformer.Get(ctx)
		deploymentInformer := deploymentinformer.Get(ctx, Selector)
		configMapInformer := configmapinformer.Get(ctx, Selector)
		kubeClient := kubeclient.Get(ctx)
		logger := logging.FromContext(ctx)

		mfclient, err := mfc.NewClient(injection.GetConfig(ctx))
		if err != nil {
			logger.Fatalw("Error creating client from injected config", zap.Error(err))
		}
		mflogger := zapr.NewLogger(logger.Named("manifestival").Desugar())
		manifest, _ := mf.ManifestFrom(mf.Slice{}, mf.UseClient(mfclient), mf.UseLogger(mflogger))

		c := &Reconciler{
			kubeClientSet:     kubeClient,
			operatorClientSet: operatorclient.Get(ctx),
			manifest:          manifest,
		}
		impl := knereconciler.NewImpl(ctx, c)
		c.extension = generator(ctx, impl)

		logger.Info("Setting up event handlers")

		knativeEventingInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

		deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterControllerGVK(v1beta1.SchemeGroupVersion.WithKind("KnativeEventing")),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})
		configMapInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterControllerGVK(v1beta1.SchemeGroupVersion.WithKind("KnativeEventing")),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})

		return impl
	}
}
