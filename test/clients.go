/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains an object which encapsulates k8s clients which are useful for e2e tests.

package test

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/client/clientset/versioned"
	operatorv1beta1 "knative.dev/operator/pkg/client/clientset/versioned/typed/operator/v1beta1"
)

// Clients holds instances of interfaces for making requests to Knative Serving.
type Clients struct {
	KubeClient kubernetes.Interface
	Dynamic    dynamic.Interface
	Operator   operatorv1beta1.OperatorV1beta1Interface
	Config     *rest.Config
}

// NewClients instantiates and returns several clientsets required for making request to the
// Knative Serving cluster specified by the combination of clusterName and configPath.
func NewClients(configPath string, clusterName string) (*Clients, error) {
	clients := &Clients{}
	cfg, err := buildClientConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	// We poll, so set our limits high.
	cfg.QPS = 100
	cfg.Burst = 200

	clients.KubeClient, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	clients.Dynamic, err = dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	clients.Operator, err = newKnativeOperatorBetaClients(cfg)
	if err != nil {
		return nil, err
	}

	clients.Config = cfg
	return clients, nil
}

func buildClientConfig(kubeConfigPath string, clusterName string) (*rest.Config, error) {
	overrides := clientcmd.ConfigOverrides{}
	// Override the cluster name if provided.
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&overrides).ClientConfig()
}

func newKnativeOperatorBetaClients(cfg *rest.Config) (operatorv1beta1.OperatorV1beta1Interface, error) {
	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return cs.OperatorV1beta1(), nil
}

func (c *Clients) KnativeServing() operatorv1beta1.KnativeServingInterface {
	return c.Operator.KnativeServings(ServingOperatorNamespace)
}

func (c *Clients) KnativeServingAll() operatorv1beta1.KnativeServingInterface {
	return c.Operator.KnativeServings(metav1.NamespaceAll)
}

func (c *Clients) KnativeEventing() operatorv1beta1.KnativeEventingInterface {
	return c.Operator.KnativeEventings(EventingOperatorNamespace)
}

func (c *Clients) KnativeEventingAll() operatorv1beta1.KnativeEventingInterface {
	return c.Operator.KnativeEventings(metav1.NamespaceAll)
}
