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

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	operatorclient "knative.dev/operator/pkg/client/injection/client"
	knativeEventinginformer "knative.dev/operator/pkg/client/injection/informers/operator/v1alpha1/knativeeventing"
	knereconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeeventing"
	"knative.dev/operator/pkg/reconciler"
	"knative.dev/operator/pkg/reconciler/common"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	deploymentinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

const (
	kcomponent = "knative-eventing"
)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	knativeEventingInformer := knativeEventinginformer.Get(ctx)
	deploymentInformer := deploymentinformer.Get(ctx)
	kubeClient := kubeclient.Get(ctx)
	logger := logging.FromContext(ctx)

	// Clean up old non-unified operator resources before even starting the controller.
	if err := reconciler.RemovePreUnifiedResources(kubeClient, "knative-eventing-operator"); err != nil {
		logger.Fatalw("Failed to remove old resources", zap.Error(err))
	}

	version := common.GetLatestRelease(kcomponent)
	manifestPath := common.RetrieveManifestPath(version, kcomponent)
	manifest, err := mfc.NewManifest(manifestPath,
		injection.GetConfig(ctx),
		mf.UseLogger(zapr.NewLogger(logger.Desugar()).WithName("manifestival")))

	if err != nil {
		logger.Fatalw("Error creating the Manifest for knative-eventing", zap.Error(err))
	}

	c := &Reconciler{
		kubeClientSet:     kubeClient,
		operatorClientSet: operatorclient.Get(ctx),
		platform:          common.GetPlatforms(ctx),
		config:            manifest,
		targetVersion:     version,
	}
	impl := knereconciler.NewImpl(ctx, c)

	logger.Info("Setting up event handlers")

	knativeEventingInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterControllerGVK(v1alpha1.SchemeGroupVersion.WithKind("KnativeEventing")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	return impl
}
