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

package knativeserving

import (
	"context"

	"github.com/go-logr/zapr"
	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"knative.dev/operator/pkg/apis/operator/v1beta1"
	operatorclient "knative.dev/operator/pkg/client/injection/client"
	knativeServinginformer "knative.dev/operator/pkg/client/injection/informers/operator/v1beta1/knativeserving"
	knsreconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1beta1/knativeserving"
	"knative.dev/operator/pkg/reconciler/common"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	deploymentinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment/filtered"
	configmapinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/configmap/filtered"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return NewExtendedController(common.NoExtension)(ctx, cmw)
}

const (
	// SelectorKey is the key of the selector for the KnativeServing resources.
	SelectorKey = "app.kubernetes.io/name"
	// SelectorValue is the value of the selector for the KnativeServing resources.
	SelectorValue = "knative-serving"
	// Selector is the selector for the KnativeServing resources.
	Selector = SelectorKey + "=" + SelectorValue
)

// NewExtendedController returns a controller extended to a specific platform
func NewExtendedController(generator common.ExtensionGenerator) injection.ControllerConstructor {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		ksInformer := knativeServinginformer.Get(ctx)
		deploymentInformer := deploymentinformer.Get(ctx, Selector)
		configMapInformer := configmapinformer.Get(ctx, Selector)
		kubeClient := kubeclient.Get(ctx)
		logger := logging.FromContext(ctx)
		logger.Infof("Remote deployments poll interval: %s", common.RemoteDeploymentsPollIntervalValue())

		restConfig := injection.GetConfig(ctx)
		mfclient, err := mfc.NewClient(restConfig)
		if err != nil {
			logger.Fatalw("Error creating client from injected config", zap.Error(err))
		}
		mflogger := zapr.NewLogger(logger.Named("manifestival").Desugar())
		manifest, _ := mf.ManifestFrom(mf.Slice{}, mf.UseClient(mfclient), mf.UseLogger(mflogger))

		clusterProvider, err := common.GetOrCreateClusterProvider(ctx, restConfig, common.ClusterProfileProviderFile())
		if err != nil {
			logger.Fatalw("Error creating cluster provider", zap.Error(err))
		}

		c := &Reconciler{
			kubeClientSet:     kubeClient,
			operatorClientSet: operatorclient.Get(ctx),
			manifest:          manifest,
			clusterProvider:   clusterProvider,
			servingLister:     ksInformer.Lister(),
		}
		impl := knsreconciler.NewImpl(ctx, c)
		c.extension = generator(ctx, impl)

		clusterProvider.RegisterListener(common.ClusterProfileListener{
			ListCRs: func(cpNamespace, cpName string) []types.NamespacedName {
				kss, err := ksInformer.Lister().List(labels.Everything())
				if err != nil {
					logger.Warnf("Failed to list KnativeServings: %v", err)
					return nil
				}
				var keys []types.NamespacedName
				for _, ks := range kss {
					ref := ks.Spec.ClusterProfileRef
					if ref != nil && ref.Namespace == cpNamespace && ref.Name == cpName {
						keys = append(keys, types.NamespacedName{
							Namespace: ks.Namespace, Name: ks.Name,
						})
					}
				}
				return keys
			},
			EnqueueKey: impl.EnqueueKey,
		})
		clusterProvider.StartInformer(ctx)

		logger.Info("Setting up event handlers")

		ksInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

		deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterControllerGVK(v1beta1.SchemeGroupVersion.WithKind("KnativeServing")),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})
		configMapInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterControllerGVK(v1beta1.SchemeGroupVersion.WithKind("KnativeServing")),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})

		return impl
	}
}
