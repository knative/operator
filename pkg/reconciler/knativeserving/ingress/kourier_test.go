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

func servingInstance(ns string, serviceType v1.ServiceType, bootstrapConfigmapName string, serviceLoadBalancerIP string) *servingv1beta1.KnativeServing {
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
					ServiceLoadBalancerIP:  serviceLoadBalancerIP,
					BootstrapConfigmapName: bootstrapConfigmapName,
				},
			},
		},
	}
}

func servingInstanceNodePorts(ns string, bootstrapConfigmapName string, httpPort int32, httpsPort int32) *servingv1beta1.KnativeServing {
	return &servingv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: ns,
		},
		Spec: servingv1beta1.KnativeServingSpec{
			Ingress: &servingv1beta1.IngressConfigs{
				Kourier: base.KourierIngressConfiguration{
					Enabled:                true,
					ServiceType:            "NodePort",
					BootstrapConfigmapName: bootstrapConfigmapName,
					HTTPPort:               httpPort,
					HTTPSPort:              httpsPort,
				},
			},
		},
	}
}

func TestTransformKourierManifest(t *testing.T) {
	tests := []struct {
		name              string
		instance          *servingv1beta1.KnativeServing
		expNamespace      string
		expServiceType    string
    expServiceLoadBalancerIP string
		expConfigMapName  string
		expNodePortsHTTP  int32
		expNodePortsHTTPS int32
		expError          error
	}{{
		name:                     "Replaces Kourier Gateway Namespace, ServiceType and bootstrap cm",
		instance:                 servingInstance(servingNamespace, "ClusterIP", "my-bootstrap", ""),
		expNamespace:             servingNamespace,
		expConfigMapName:         "my-bootstrap",
		expServiceType:           "ClusterIP",
	}, {
		name:                     "Use Kourier default service type",
		instance:                 servingInstance(servingNamespace, "" /* empty service type */, "", ""),
		expNamespace:             servingNamespace,
		expConfigMapName:         kourierDefaultVolumeName,
		expServiceType:           "LoadBalancer", // kourier GW default service type
		expServiceLoadBalancerIP: "",
	}, {
		name:                     "Sets Kourier Gateway ServiceLoadBalancerIP",
		instance:                 servingInstance(servingNamespace, "" /* empty service type */, "", "1.2.3.4"),
		expNamespace:             servingNamespace,
		expConfigMapName:         kourierDefaultVolumeName,
		expServiceType:           "LoadBalancer",
		expServiceLoadBalancerIP: "1.2.3.4",
	}, {
		name:                     "Use ServiceLoadBalancerIP with unsupported service type",
		instance:                 servingInstance(servingNamespace, "ClusterIP", "", "1.2.3.4"),
		expNamespace:             servingNamespace,
		expConfigMapName:         kourierDefaultVolumeName,
		expServiceType:           "ClusterIP",
		expServiceLoadBalancerIP: "1.2.3.4",
		expError:                 fmt.Errorf("cannot configure LoadBalancerIP for service type \"ClusterIP\""),
	}, {
		name:             "Use unknown service type",
		instance:         servingInstance(servingNamespace, "Foo", ""),
		expNamespace:     servingNamespace,
		expServiceType:   "Foo",
		expConfigMapName: kourierDefaultVolumeName,
		expError:         fmt.Errorf("unknown service type \"Foo\""),
	}, {
		name:              "Use NodePort service type",
		instance:          servingInstanceNodePorts(servingNamespace, "", 30001, 30002),
		expNamespace:      servingNamespace,
		expServiceType:    "NodePort",
		expNodePortsHTTP:  30001,
		expNodePortsHTTPS: 30002,
		expConfigMapName:  kourierDefaultVolumeName,
	}, {
		name:              "Use NodePort service type with unset HTTP Port",
		instance:          servingInstanceNodePorts(servingNamespace, "", 0, 30002),
		expNamespace:      servingNamespace,
		expServiceType:    "NodePort",
		expNodePortsHTTP:  0,
		expNodePortsHTTPS: 30002,
		expConfigMapName:  kourierDefaultVolumeName,
	}, {
		name:              "Use NodePort service type with unset HTTPS Port",
		instance:          servingInstanceNodePorts(servingNamespace, "", 30001, 0),
		expNamespace:      servingNamespace,
		expServiceType:    "NodePort",
		expNodePortsHTTP:  30001,
		expNodePortsHTTPS: 0,
		expConfigMapName:  kourierDefaultVolumeName,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.New()
			manifest, err := mf.NewManifest("testdata/kodata/ingress/1.9/kourier/kourier.yaml", mf.UseClient(client))
			if err != nil {
				t.Fatalf("Failed to read manifest: %v", err)
			}

			manifest, err = manifest.Transform(replaceGatewayNamespace())
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}

			manifest, err = manifest.Transform(configureGatewayService(tt.instance))
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
				verifyGatewayServiceLoadBalancerIP(t, &u, tt.expServiceLoadBalancerIP)
				verifyGatewayServiceTypeNodePortHTTP(t, &u, tt.expNodePortsHTTP)
				verifyGatewayServiceTypeNodePortHTTPS(t, &u, tt.expNodePortsHTTPS)
				verifyBootstrapVolumeName(t, &u, tt.expConfigMapName)
			}
		})
	}
}

func verifyGatewayServiceTypeNodePortHTTP(t *testing.T, u *unstructured.Unstructured, expHTTPPort int32) {
	if u.GetKind() == "Service" && u.GetName() == kourierGatewayServiceName {
		svc := &v1.Service{}
		err := scheme.Scheme.Convert(u, svc, nil)
		util.AssertEqual(t, err, nil)
		svcPorts := svc.Spec.Ports
		var resultPort int32
		for _, port := range svcPorts {
			if port.Name != "https" {
				resultPort = port.NodePort
			}
		}
		util.AssertDeepEqual(t, resultPort, expHTTPPort)
	}
}

func verifyGatewayServiceTypeNodePortHTTPS(t *testing.T, u *unstructured.Unstructured, expHTTPSPort int32) {
	if u.GetKind() == "Service" && u.GetName() == kourierGatewayServiceName {
		svc := &v1.Service{}
		err := scheme.Scheme.Convert(u, svc, nil)
		util.AssertEqual(t, err, nil)
		svcPorts := svc.Spec.Ports
		var resultPort int32
		for _, port := range svcPorts {
			if port.Name == "https" {
				resultPort = port.NodePort
			}
		}
		util.AssertDeepEqual(t, resultPort, expHTTPSPort)
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

func verifyGatewayServiceLoadBalancerIP(t *testing.T, u *unstructured.Unstructured, expServiceLoadBalancerIP string) {
	if u.GetKind() == "Service" && u.GetName() == kourierGatewayServiceName {
		svc := &v1.Service{}
		err := scheme.Scheme.Convert(u, svc, nil)
		util.AssertEqual(t, err, nil)
		svcLoadBalancerIP := svc.Spec.LoadBalancerIP
		util.AssertDeepEqual(t, svcLoadBalancerIP, expServiceLoadBalancerIP)
	}
}

// removeProviderLabels removes labels. This util is used for tests without provider label.
func removeLabels() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		u.SetLabels(map[string]string{})
		return nil
	}
}
