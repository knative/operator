/*
Copyright 2022 The Knative Authors

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

package v1alpha1

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestKnativeServingConvertTo(t *testing.T) {
	source := &KnativeServing{
		Spec: KnativeServingSpec{
			CommonSpec: base.CommonSpec{
				Version: "1.2",
				Resources: []base.ResourceRequirementsOverride{{
					Container: "webhook",
					ResourceRequirements: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
							corev1.ResourceMemory: resource.MustParse("999Mi")},
						Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
							corev1.ResourceMemory: resource.MustParse("999Mi")},
					},
				}},
			},
			Ingress: &IngressConfigs{
				Istio: base.IstioIngressConfiguration{
					Enabled: true,
				},
				Contour: base.ContourIngressConfiguration{
					Enabled: false,
				},
			},
			ControllerCustomCerts: base.CustomCerts{
				Type: "test-type",
				Name: "test-name",
			},
		},
	}
	sink := &v1beta1.KnativeServing{}
	expectedResourceRequirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
			corev1.ResourceMemory: resource.MustParse("999Mi")},
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
			corev1.ResourceMemory: resource.MustParse("999Mi")},
	}
	err := source.ConvertTo(context.Background(), sink)
	util.AssertEqual(t, err, nil)
	util.AssertEqual(t, sink.GetSpec().GetVersion(), "1.2")
	util.AssertEqual(t, sink.Spec.ControllerCustomCerts.Name, "test-name")
	util.AssertEqual(t, sink.Spec.ControllerCustomCerts.Type, "test-type")
	util.AssertEqual(t, sink.Spec.Ingress.Istio.Enabled, true)
	util.AssertEqual(t, sink.Spec.Ingress.Contour.Enabled, false)
	util.AssertEqual(t, sink.Spec.DeploymentOverride[0].Resources[0].Container, "webhook")
	util.AssertDeepEqual(t, sink.Spec.DeploymentOverride[0].Resources[0].ResourceRequirements,
		expectedResourceRequirements)
}

func TestTestKnativeServingConvertFrom(t *testing.T) {
	ke := &KnativeServing{}
	source := &v1beta1.KnativeServing{
		Spec: v1beta1.KnativeServingSpec{
			CommonSpec: base.CommonSpec{
				Version: "1.2",
			},
			Ingress: &v1beta1.IngressConfigs{
				Istio: base.IstioIngressConfiguration{
					Enabled: true,
				},
				Contour: base.ContourIngressConfiguration{
					Enabled: false,
				},
			},
			ControllerCustomCerts: base.CustomCerts{
				Type: "test-type",
				Name: "test-name",
			},
		},
	}
	err := ke.ConvertFrom(context.Background(), source)
	util.AssertEqual(t, err, nil)
	util.AssertEqual(t, ke.GetSpec().GetVersion(), "1.2")
	util.AssertEqual(t, ke.Spec.ControllerCustomCerts.Name, "test-name")
	util.AssertEqual(t, ke.Spec.ControllerCustomCerts.Type, "test-type")
	util.AssertEqual(t, ke.Spec.Ingress.Istio.Enabled, true)
	util.AssertEqual(t, ke.Spec.Ingress.Contour.Enabled, false)
}
