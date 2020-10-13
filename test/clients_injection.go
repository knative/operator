/*
Copyright 2019 The Knative Authors

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

// This file contains an object which encapsulates k8s clients which are useful for e2e tests.

package test

import (
	"context"

	operatorclient "knative.dev/operator/pkg/client/injection/client"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/clients/dynamicclient"
)

// NewClientsFromCtx instantiates and returns several clientsets required for making request to the
// Knative Serving cluster specified by the combination of clusterName and configPath.
func NewClientsFromCtx(ctx context.Context) (*Clients, error) {
	clients := &Clients{
		Kube:     kubeclient.Get(ctx),
		Dynamic:  dynamicclient.Get(ctx),
		Operator: operatorclient.Get(ctx).OperatorV1alpha1(),
		Config:   injection.GetConfig(ctx),
	}
	return clients, nil
}
