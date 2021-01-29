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

package ingress

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

const (
	DEPLOY_NAME        = "3scale-kourier-control"
	Kourier_GATEWAY_NS = "kourier-system"
	Expected_NS        = "knative-serving"
)

func TestReplaceKourierGWNamespace(t *testing.T) {
	tests := []struct {
		name           string
		deploymentName string
		labels         map[string]string
		ns             string
		expectedNS     string
		expected       []v1.Container
	}{{
		name:           "Replaces Kourier Gateway Namespace",
		deploymentName: DEPLOY_NAME,
		labels: map[string]string{
			"networking.knative.dev/ingress-provider": "kourier",
		},
		ns:         Kourier_GATEWAY_NS,
		expectedNS: Expected_NS,
		expected: []v1.Container{{
			Name: DEPLOY_NAME,
			Env:  []v1.EnvVar{{Name: KourierGatewayNSEnvVarKey, Value: Expected_NS}},
		}},
	}, {
		name:           "Do Not Replace Kourier Gateway Namespace without the ingress label",
		deploymentName: DEPLOY_NAME,
		labels:         map[string]string{},
		ns:             Kourier_GATEWAY_NS,
		expectedNS:     Expected_NS,
		expected: []v1.Container{{
			Name: DEPLOY_NAME,
			Env:  []v1.EnvVar{{Name: KourierGatewayNSEnvVarKey, Value: Kourier_GATEWAY_NS}},
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := makeUnstructuredDeployment(t, tt.deploymentName, tt.ns, tt.labels)
			replaceKourierGWNamespace(tt.expectedNS)(u)
			deployment := &appsv1.Deployment{}
			err := scheme.Scheme.Convert(u, deployment, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, deployment.Spec.Template.Spec.Containers, tt.expected)
		})
	}
}

func makeUnstructuredDeployment(t *testing.T, name, ns string, labels map[string]string) *unstructured.Unstructured {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name: name,
						Env:  []v1.EnvVar{{Name: KourierGatewayNSEnvVarKey, Value: ns}},
					}},
				},
			},
		},
	}
	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(d, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured Deployment: %v, err: %v", d, err)
	}
	return result
}
