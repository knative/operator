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
	"fmt"
	"testing"

	mf "github.com/manifestival/manifestival"
	fake "github.com/manifestival/manifestival/fake"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

const servingNamespace = "knative-serving"

func servingInstance(ns string, serviceType v1.ServiceType, bootstrapConfigmapName string) *servingv1beta1.KnativeServing {
	return &servingv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: ns,
		},
		Spec: servingv1beta1.KnativeServingSpec{
			Ingress: &servingv1beta1.IngressConfigs{
				Kourier: base.KourierIngressConfiguration{
					Enabled:                true,
					ServiceType:            serviceType,
					BootstrapConfigmapName: bootstrapConfigmapName,
				},
			},
		},
	}
}

func TestTransformKourierManifest(t *testing.T) {
	tests := []struct {
		name             string
		instance         *servingv1beta1.KnativeServing
		dropLabel        bool
		expNamespace     string
		expServiceType   string
		expConfigMapName string
		expError         error
	}{{
		name:             "Replaces Kourier Gateway Namespace, ServiceType and bootstrap cm",
		instance:         servingInstance(servingNamespace, "ClusterIP", "my-bootstrap"),
		expNamespace:     servingNamespace,
		expConfigMapName: "my-bootstrap",
		expServiceType:   "ClusterIP",
	}, {
		name:             "Use Kourier default service type",
		instance:         servingInstance(servingNamespace, "" /* empty service type */, ""),
		expNamespace:     servingNamespace,
		expConfigMapName: kourierDefaultVolumeName,
		expServiceType:   "LoadBalancer", // kourier GW default service type
	}, {
		name:             "Use unsupported service type",
		instance:         servingInstance(servingNamespace, "ExternalName", ""),
		expNamespace:     servingNamespace,
		expServiceType:   "ExternalName",
		expConfigMapName: kourierDefaultVolumeName,
		expError:         fmt.Errorf("unsupported service type \"ExternalName\""),
	}, {
		name:             "Use unknown service type",
		instance:         servingInstance(servingNamespace, "Foo", ""),
		expNamespace:     servingNamespace,
		expServiceType:   "Foo",
		expConfigMapName: kourierDefaultVolumeName,
		expError:         fmt.Errorf("unknown service type \"Foo\""),
	}, {
		name:             "Do not transform without the ingress provier label",
		dropLabel:        true,
		instance:         servingInstance(servingNamespace, "ClusterIP", "my-bootstrap"),
		expNamespace:     "kourier-system", // kourier default namespace
		expConfigMapName: kourierDefaultVolumeName,
		expServiceType:   "LoadBalancer", // kourier GW default service type
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.New()
			manifest, err := mf.NewManifest("testdata/kodata/ingress/0.20/kourier.yaml", mf.UseClient(client))
			if err != nil {
				t.Fatalf("Failed to read manifest: %v", err)
			}

			if tt.dropLabel {
				manifest, err = manifest.Transform(removeLabels())
				if err != nil {
					t.Fatalf("Failed to transform manifest: %v", err)
				}
			}

			manifest, err = manifest.Transform(replaceGWNamespace())
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}

			manifest, err = manifest.Transform(configureGWServiceType(tt.instance))
			if err != nil {
				util.AssertEqual(t, err.Error(), tt.expError.Error())
			} else {
				util.AssertEqual(t, err, tt.expError)
			}

			manifest, err = manifest.Transform(configureBootstrapConfigMap(tt.instance))
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}

			for _, u := range manifest.Resources() {
				verifyControllerNamespace(t, &u, tt.expNamespace)
				verifyGatewayServiceType(t, &u, tt.expServiceType)
				verifyBootstrapVolumeName(t, &u, tt.expConfigMapName)
			}
		})
	}
}

func verifyControllerNamespace(t *testing.T, u *unstructured.Unstructured, expNamespace string) {
	if u.GetKind() == "Deployment" && kourierControllerDeploymentNames.Has(u.GetName()) {
		deployment := &appsv1.Deployment{}
		err := scheme.Scheme.Convert(u, deployment, nil)
		util.AssertEqual(t, err, nil)
		envs := deployment.Spec.Template.Spec.Containers[0].Env
		env := ""
		for i := range envs {
			if envs[i].Name == kourierGatewayNSEnvVarKey {
				env = envs[i].Value
			}
		}
		util.AssertDeepEqual(t, env, expNamespace)
	}
}

func verifyBootstrapVolumeName(t *testing.T, u *unstructured.Unstructured, expConfigMapName string) {
	if u.GetKind() == "Deployment" && u.GetName() == kourierGatewayDeploymentNames {
		deployment := &appsv1.Deployment{}
		err := scheme.Scheme.Convert(u, deployment, nil)
		util.AssertEqual(t, err, nil)
		configMapName := deployment.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name
		util.AssertDeepEqual(t, configMapName, expConfigMapName)
	}
}

func verifyGatewayServiceType(t *testing.T, u *unstructured.Unstructured, expServiceType string) {
	if u.GetKind() == "Service" && u.GetName() == kourierGatewayServiceName {
		svc := &v1.Service{}
		err := scheme.Scheme.Convert(u, svc, nil)
		util.AssertEqual(t, err, nil)
		svcType := svc.Spec.Type
		util.AssertDeepEqual(t, string(svcType), expServiceType)
	}
}

// removeProviderLabels removes labels. This util is used for tests without provider label.
func removeLabels() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		u.SetLabels(map[string]string{})
		return nil
	}
}
