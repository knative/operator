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
package common

import (
	"testing"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

var log = zap.NewNop().Sugar()

type customCertsTest struct {
	name         string
	input        servingv1alpha1.CustomCerts
	expectError  bool
	expectSource *v1.VolumeSource
}

var customCertsTests = []customCertsTest{
	{
		name: "FromSecret",
		input: servingv1alpha1.CustomCerts{
			Type: "Secret",
			Name: "my-secret",
		},
		expectError: false,
		expectSource: &v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: "my-secret",
			},
		},
	},
	{
		name: "FromConfigMap",
		input: servingv1alpha1.CustomCerts{
			Type: "ConfigMap",
			Name: "my-map",
		},
		expectError: false,
		expectSource: &v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "my-map",
				},
			},
		},
	},
	{
		name:        "NoCerts",
		input:       servingv1alpha1.CustomCerts{},
		expectError: false,
	},
	{
		name: "InvalidType",
		input: servingv1alpha1.CustomCerts{
			Type: "invalid",
		},
		expectError: true,
	},
	{
		name: "MissingName",
		input: servingv1alpha1.CustomCerts{
			Type: "Secret",
		},
		expectError: true,
	},
}

func TestOnlyTransformCustomCertsForController(t *testing.T) {
	before := util.MakeDeployment("not-controller", v1.PodSpec{
		Containers: []v1.Container{{
			Name: "definitely-not-controller",
		}},
	})
	instance := &servingv1alpha1.KnativeServing{
		Spec: servingv1alpha1.KnativeServingSpec{
			ControllerCustomCerts: servingv1alpha1.CustomCerts{
				Type: "Secret",
				Name: "my-secret",
			},
		},
	}
	customCertsTransform := CustomCertsTransform(instance, log)
	unstructured := util.MakeUnstructured(t, before)
	err := customCertsTransform(&unstructured)
	util.AssertEqual(t, err, nil)
	after := &appsv1.Deployment{}
	err = scheme.Scheme.Convert(&unstructured, after, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, after.Spec, before.Spec)
}

func TestCustomCertsTransform(t *testing.T) {
	for _, tt := range customCertsTests {
		t.Run(tt.name, func(t *testing.T) {
			runCustomCertsTransformTest(t, &tt)
		})
	}
}

func runCustomCertsTransformTest(t *testing.T, tt *customCertsTest) {
	unstructured := util.MakeUnstructured(t, util.MakeDeployment("controller", v1.PodSpec{
		Containers: []v1.Container{{
			Name: "controller",
		}},
	}))
	instance := &servingv1alpha1.KnativeServing{
		Spec: servingv1alpha1.KnativeServingSpec{
			ControllerCustomCerts: tt.input,
		},
	}
	customCertsTransform := CustomCertsTransform(instance, log)
	err := customCertsTransform(&unstructured)
	if tt.expectError && err == nil {
		t.Fatal("Transformer should've returned an error and did not")
	}
	validateCustomCertsTransform(t, tt, &unstructured)
}

func validateCustomCertsTransform(t *testing.T, tt *customCertsTest, u *unstructured.Unstructured) {
	deployment := &appsv1.Deployment{}
	err := scheme.Scheme.Convert(u, deployment, nil)
	util.AssertEqual(t, err, nil)
	spec := deployment.Spec.Template.Spec
	if tt.expectSource != nil {
		util.AssertEqual(t, spec.Volumes[0].Name, customCertsNamePrefix+tt.input.Name)
		util.AssertDeepEqual(t, &spec.Volumes[0].VolumeSource, tt.expectSource)
		util.AssertDeepEqual(t, spec.Containers[0].Env[0], v1.EnvVar{
			Name:  customCertsEnvName,
			Value: customCertsMountPath,
		})
		util.AssertDeepEqual(t, spec.Containers[0].VolumeMounts[0], v1.VolumeMount{
			Name:      customCertsNamePrefix + tt.input.Name,
			MountPath: customCertsMountPath,
		})
	} else {
		util.AssertEqual(t, len(spec.Volumes), 0)
		util.AssertEqual(t, len(spec.Containers[0].Env), 0)
		util.AssertEqual(t, len(spec.Containers[0].VolumeMounts), 0)
	}
}
