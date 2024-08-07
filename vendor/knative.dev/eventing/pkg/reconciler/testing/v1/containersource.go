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

package testing

import (
	"context"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	v1 "knative.dev/eventing/pkg/apis/sources/v1"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ContainerSourceOption enables further configuration of a ContainerSource.
type ContainerSourceOption func(*v1.ContainerSource)

// NewContainerSource creates a v1 ContainerSource with ContainerSourceOptions
func NewContainerSource(name, namespace string, o ...ContainerSourceOption) *v1.ContainerSource {
	c := &v1.ContainerSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, opt := range o {
		opt(c)
	}
	c.SetDefaults(context.Background())
	return c
}

func WithContainerSourceUID(uid types.UID) ContainerSourceOption {
	return func(s *v1.ContainerSource) {
		s.UID = uid
	}
}

// WithInitContainerSourceConditions initializes the ContainerSource's conditions.
func WithInitContainerSourceConditions(s *v1.ContainerSource) {
	s.Status.InitializeConditions()
}

func WithContainerSourcePropagateReceiveAdapterStatus(d *appsv1.Deployment) ContainerSourceOption {
	return func(s *v1.ContainerSource) {
		s.Status.PropagateReceiveAdapterStatus(d)
	}
}

func WithContainerSourcePropagateSinkbindingStatus(status *v1.SinkBindingStatus) ContainerSourceOption {
	return func(s *v1.ContainerSource) {
		s.Status.PropagateSinkBindingStatus(status)
	}
}

func WithContainerSourceSpec(spec v1.ContainerSourceSpec) ContainerSourceOption {
	return func(c *v1.ContainerSource) {
		c.Spec = spec
	}
}

func WithContainerSourceStatusObservedGeneration(generation int64) ContainerSourceOption {
	return func(c *v1.ContainerSource) {
		c.Status.ObservedGeneration = generation
	}
}

func WithContainerSourceObjectMetaGeneration(generation int64) ContainerSourceOption {
	return func(c *v1.ContainerSource) {
		c.ObjectMeta.Generation = generation
	}
}

func WithContainerUnobservedGeneration() ContainerSourceOption {
	return func(c *v1.ContainerSource) {
		condSet := c.GetConditionSet()
		condSet.Manage(&c.Status).MarkUnknown(
			condSet.GetTopLevelConditionType(), "NewObservedGenFailure", "unsuccessfully observed a new generation")
	}
}

func WithContainerSourceOIDCServiceAccountName(name string) ContainerSourceOption {
	return func(c *v1.ContainerSource) {
		if c.Status.Auth == nil {
			c.Status.Auth = &duckv1.AuthStatus{}
		}

		c.Status.Auth.ServiceAccountName = &name
	}
}
