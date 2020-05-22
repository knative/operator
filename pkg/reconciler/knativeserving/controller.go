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
	"fmt"
	"go.uber.org/zap"

	operatorclient "knative.dev/operator/pkg/client/injection/client"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"

	"k8s.io/client-go/tools/cache"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	knativeServinginformer "knative.dev/operator/pkg/client/injection/informers/operator/v1alpha1/knativeserving"
	knsreconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeserving"
	"knative.dev/operator/pkg/reconciler"
	"knative.dev/operator/pkg/reconciler/common"
	servingcommon "knative.dev/operator/pkg/reconciler/knativeserving/common"
	deploymentinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

const (
	controllerAgentName = "knativeserving-controller"
)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	knativeServingInformer := knativeServinginformer.Get(ctx)
	deploymentInformer := deploymentinformer.Get(ctx)
	kubeClient := kubeclient.Get(ctx)
	logger := logging.FromContext(ctx)

	// Clean up old non-unified operator resources before even starting the controller.
	if err := reconciler.RemovePreUnifiedResources(kubeClient, "knative-serving-operator"); err != nil {
		logger.Fatalw("Failed to remove old resources", zap.Error(err))
	}

	statsReporter, err := servingcommon.NewStatsReporter(controllerAgentName)
	if err != nil {
		logger.Fatal(err)
	}

	c := &Reconciler{
		kubeClientSet:     kubeClient,
		operatorClientSet: operatorclient.Get(ctx),
		platform:          common.GetPlatforms(ctx),
		config:		       injection.GetConfig(ctx),
	}
	impl := knsreconciler.NewImpl(ctx, c)

	logger.Info("Setting up event handlers")

	knativeServingInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterControllerGVK(v1alpha1.SchemeGroupVersion.WithKind("KnativeServing")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	// Reporting statistics on KnativeServing events.
	knativeServingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(newObj interface{}) {
			new := newObj.(*servingv1alpha1.KnativeServing)
			if new.Generation == 1 {
				statsReporter.ReportKnativeservingChange(key(new), "creation")
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			old := oldObj.(*servingv1alpha1.KnativeServing)
			new := newObj.(*servingv1alpha1.KnativeServing)
			if old.Generation < new.Generation {
				statsReporter.ReportKnativeservingChange(key(new), "edit")
			}
		},
		DeleteFunc: func(oldObj interface{}) {
			old := oldObj.(*servingv1alpha1.KnativeServing)
			statsReporter.ReportKnativeservingChange(key(old), "deletion")
		},
	})

	return impl
}

func key(ks *servingv1alpha1.KnativeServing) string {
	return fmt.Sprintf("%s/%s", ks.Namespace, ks.Name)
}
