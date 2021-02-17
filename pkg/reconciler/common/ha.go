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

package common

import (
	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	v1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const (
	configMapName           = "config-leader-election"
	enabledComponentsKey    = "enabledComponents"
	servingComponentsValue  = "controller,hpaautoscaler,certcontroller,istiocontroller,nscontroller"
	eventingComponentsValue = "eventing-controller,sugar-controller,imc-controller,imc-dispatcher,mt-broker-controller"
)

func haSupport(obj v1alpha1.KComponent) sets.String {
	deploymentNames := sets.NewString(
		"controller",
		"autoscaler-hpa",
		"networking-certmanager",
		"networking-ns-cert",
		"networking-istio",
		"eventing-controller",
		"sugar-controller",
		"imc-controller",
		"imc-dispatcher",
		"mt-broker-controller",
	)

	// HA for autoscaler is supported since v0.19+.
	version := obj.GetStatus().GetVersion()
	if semver.Compare(SanitizeSemver(version), "v0.19") >= 0 {
		deploymentNames.Insert("autoscaler")
	}

	return deploymentNames
}

// HighAvailabilityTransform mutates configmaps and replicacounts of certain
// controllers when HA control plane is specified.
func HighAvailabilityTransform(obj v1alpha1.KComponent, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// stash the HA object
		ha := obj.GetSpec().GetHighAvailability()
		if ha == nil {
			return nil
		}

		// Transform the leader election config.
		if u.GetKind() == "ConfigMap" && u.GetName() == "config-leader-election" {
			data, ok, err := unstructured.NestedStringMap(u.UnstructuredContent(), "data")
			if err != nil {
				return nil
			}
			if !ok {
				data = map[string]string{}
			}

			if _, ok := obj.(*v1alpha1.KnativeServing); ok {
				data[enabledComponentsKey] = servingComponentsValue
			}
			if _, ok := obj.(*v1alpha1.KnativeEventing); ok {
				data[enabledComponentsKey] = eventingComponentsValue
			}
			if err := unstructured.SetNestedStringMap(u.Object, data, "data"); err != nil {
				return err
			}
		}

		replicas := int64(ha.Replicas)

		// Transform deployments that support HA.
		if u.GetKind() == "Deployment" && haSupport(obj).Has(u.GetName()) {
			if err := unstructured.SetNestedField(u.Object, replicas, "spec", "replicas"); err != nil {
				return err
			}
		}

		if u.GetKind() == "HorizontalPodAutoscaler" {
			min, _, err := unstructured.NestedInt64(u.Object, "spec", "minReplicas")
			if err != nil {
				return err
			}
			// Do nothing if the HPA ships with even more replicas out of the box.
			if min > replicas {
				return nil
			}

			if err := unstructured.SetNestedField(u.Object, replicas, "spec", "minReplicas"); err != nil {
				return err
			}
		}

		return nil
	}
}
