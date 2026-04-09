/*
Copyright 2025 The Knative Authors

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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"

	"knative.dev/pkg/logging"

	clusterinventoryv1alpha1 "sigs.k8s.io/cluster-inventory-api/apis/v1alpha1"
)

const clusterProfileResyncPeriod = 10 * time.Minute

func (c *ClusterProvider) StartInformer(ctx context.Context) {
	c.mu.Lock()
	if c.informerStarted {
		c.mu.Unlock()
		return
	}
	c.informerStarted = true
	c.mu.Unlock()

	logger := logging.FromContext(ctx)

	_, err := c.ciClient.ApisV1alpha1().ClusterProfiles("").List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		logger.Infof("ClusterProfile API not available (%v); multi-cluster informer disabled", err)
		return
	}

	_, informer := toolscache.NewInformerWithOptions(toolscache.InformerOptions{
		ListerWatcher: &toolscache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				return c.ciClient.ApisV1alpha1().ClusterProfiles("").List(c.controllerCtx, opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				return c.ciClient.ApisV1alpha1().ClusterProfiles("").Watch(c.controllerCtx, opts)
			},
		},
		ObjectType:   &clusterinventoryv1alpha1.ClusterProfile{},
		ResyncPeriod: clusterProfileResyncPeriod,
		Handler: toolscache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cp, ok := obj.(*clusterinventoryv1alpha1.ClusterProfile)
				if !ok {
					return
				}
				c.notifyListeners(cp.Namespace, cp.Name)
			},
			UpdateFunc: func(_, newObj interface{}) {
				cp, ok := newObj.(*clusterinventoryv1alpha1.ClusterProfile)
				if !ok {
					return
				}
				c.notifyListeners(cp.Namespace, cp.Name)
			},
			DeleteFunc: func(obj interface{}) {
				cp, ok := obj.(*clusterinventoryv1alpha1.ClusterProfile)
				if !ok {
					tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
					if !ok {
						return
					}
					cp, ok = tombstone.Obj.(*clusterinventoryv1alpha1.ClusterProfile)
					if !ok {
						return
					}
				}
				c.Remove(cp.Namespace + "/" + cp.Name)
				c.notifyListeners(cp.Namespace, cp.Name)
			},
		},
	})

	go informer.Run(c.controllerCtx.Done())

	syncCtx, syncCancel := context.WithTimeout(c.controllerCtx, 10*time.Second)
	defer syncCancel()
	if !toolscache.WaitForCacheSync(syncCtx.Done(), informer.HasSynced) {
		logger.Warn("ClusterProfile informer cache sync timed out")
	} else {
		logger.Info("ClusterProfile informer cache synced")
	}
}
