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

// knativeserving.go provides methods to perform actions on the KnativeServing resource.

package resources

import (
	"context"
	"fmt"
	"os"
	"strings"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	servingv1beta1 "knative.dev/operator/pkg/client/clientset/versioned/typed/operator/v1beta1"
	"knative.dev/operator/test"
	"knative.dev/pkg/test/logging"
)

// WaitForKnativeServingState polls the status of the KnativeServing called name
// from client every `interval` until `inState` returns `true` indicating it
// is done, returns an error or timeout.
func WaitForKnativeServingState(clients servingv1beta1.KnativeServingInterface, name1 string,
	inState func(s *v1beta1.KnativeServing, err error) (bool, error)) (*v1beta1.KnativeServing, error) {
	span := logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForKnativeServingState/%s/%s", name1, "KnativeServingIsReady"))
	defer span.End()

	var lastState *v1beta1.KnativeServing
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err := clients.Get(context.TODO(), name1, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("knativeserving %s is not in desired state, got: %+v: %w", name1, lastState, waitErr)
	}
	return lastState, nil
}

// EnsureKnativeServingExists creates a KnativeServing with the name names.KnativeServing under the namespace names.Namespace, if it does not exist.
func EnsureKnativeServingExists(clients servingv1beta1.KnativeServingInterface, names test.ResourceNames) (*v1beta1.KnativeServing, error) {
	// If this function is called by the upgrade tests, we only create the custom resource, if it does not exist.
	ks, err := clients.Get(context.TODO(), names.KnativeServing, metav1.GetOptions{})
	if apierrs.IsNotFound(err) {
		ks := &v1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      names.KnativeServing,
				Namespace: names.Namespace,
			},
		}
		configureIngressClass(&ks.Spec)
		return clients.Create(context.TODO(), ks, metav1.CreateOptions{})
	}
	return ks, err
}

// WaitForConfigMap takes a condition function that evaluates ConfigMap data
func WaitForConfigMap(name string, client kubernetes.Interface, fn func(map[string]string) bool) error {
	ns, cm, _ := cache.SplitMetaNamespaceKey(name)
	return wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		cm, err := client.CoreV1().ConfigMaps(ns).Get(context.TODO(), cm, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return fn(cm.Data), nil
	})
}

// IsKnativeServingReady will check the status conditions of the KnativeServing and return true if the KnativeServing is ready.
func IsKnativeServingReady(s *v1beta1.KnativeServing, err error) (bool, error) {
	return s.Status.IsReady(), err
}

// IsDeploymentAvailable will check the status conditions of the deployment and return true if the deployment is available.
func IsDeploymentAvailable(d *v1.Deployment) (bool, error) {
	return getDeploymentStatus(d) == "True", nil
}

func getDeploymentStatus(d *v1.Deployment) corev1.ConditionStatus {
	for _, dc := range d.Status.Conditions {
		if dc.Type == "Available" {
			return dc.Status
		}
	}
	return "unknown"
}

func getTestKSOperatorCRSpec() v1beta1.KnativeServingSpec {
	spec := v1beta1.KnativeServingSpec{
		CommonSpec: base.CommonSpec{
			Config: base.ConfigMapData{
				DefaultsConfigKey: {
					"revision-timeout-seconds": "200",
				},
				LoggingConfigKey: {
					"loglevel.controller": "debug",
					"loglevel.autoscaler": "debug",
				},
			},
		},
	}
	configureIngressClass(&spec)
	return spec
}

func configureIngressClass(spec *v1beta1.KnativeServingSpec) {
	ingressClass := "istio.ingress.networking.knative.dev"
	if ingressClassOverride := os.Getenv("INGRESS_CLASS"); ingressClassOverride != "" {
		ingressClass = ingressClassOverride
	}
	var istioEnabled, contourEnabled, kourierEnabled bool
	switch strings.Split(ingressClass, ".")[0] {
	case "istio":
		istioEnabled = true
	case "contour":
		contourEnabled = true
	case "kourier":
		kourierEnabled = true
	}

	if !istioEnabled {
		spec.Ingress = &v1beta1.IngressConfigs{
			Istio:   base.IstioIngressConfiguration{Enabled: istioEnabled},
			Contour: base.ContourIngressConfiguration{Enabled: contourEnabled},
			Kourier: base.KourierIngressConfiguration{Enabled: kourierEnabled},
		}

		if spec.CommonSpec.Config == nil {
			spec.CommonSpec.Config = base.ConfigMapData{"network": {"ingress.class": ingressClass}}
		} else {
			spec.CommonSpec.Config["network"] = map[string]string{"ingress.class": ingressClass}
		}
	}
}
