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
	"flag"
	"os"
	"path/filepath"

	servingclient "knative.dev/operator/pkg/client/injection/client"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"

	"github.com/go-logr/zapr"
	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	"k8s.io/client-go/tools/cache"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	knativeServinginformer "knative.dev/operator/pkg/client/injection/informers/operator/v1alpha1/knativeserving"
	knsreconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeserving"
	"knative.dev/operator/pkg/reconciler"
	"knative.dev/operator/pkg/reconciler/common"
	deploymentinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

const (
	controllerAgentName = "knativeserving-controller"
)

var (
	MasterURL  = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	Kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	knativeServingInformer := knativeServinginformer.Get(ctx)
	deploymentInformer := deploymentinformer.Get(ctx)
	logger := logging.FromContext(ctx)

	statsReporter, err := reconciler.NewStatsReporter(controllerAgentName)
	if err != nil {
		logger.Fatal(err)
	}

	c := &Reconciler{
		kubeClientSet:           kubeclient.Get(ctx),
		knativeServingClientSet: servingclient.Get(ctx),
		statsReporter:           statsReporter,
		knativeServingLister:    knativeServingInformer.Lister(),
		servings:                map[string]int64{},
		platform:                common.GetPlatforms(ctx),
	}

	koDataDir := os.Getenv("KO_DATA_PATH")

	cfg, err := sharedmain.GetConfig(*MasterURL, *Kubeconfig)
	if err != nil {
		logger.Error(err, "Error building kubeconfig")
	}

	config, err := mfc.NewManifest(filepath.Join(koDataDir, "knative-serving/"),
		cfg,
		mf.UseLogger(zapr.NewLogger(logger.Desugar()).WithName("manifestival")))
	if err != nil {
		logger.Error(err, "Error creating the Manifest for knative-serving")
		os.Exit(1)
	}

	c.config = config
	impl := knsreconciler.NewImpl(ctx, c)

	logger.Info("Setting up event handlers")

	knativeServingInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("KnativeServing")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	return impl
}
