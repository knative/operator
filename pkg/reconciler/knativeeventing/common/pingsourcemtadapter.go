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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type unstructuredGetter interface {
	Get(obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
}

// PingsourceMTAadapterTransform keeps the number of replicas and the env vars, if the deployment
// pingsource-mt-adapter exists in the cluster.
func PingsourceMTAadapterTransform(client unstructuredGetter) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && u.GetName() == "pingsource-mt-adapter" {
			current, err := client.Get(u)
			if errors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			return setReplicaEnvVars(u, current)
		}
		return nil
	}
}

func setReplicaEnvVars(u, current *unstructured.Unstructured) error {
	numReplicas, found := nestedInt64OrFloat64(current.Object, "spec", "replicas")
	if found {
		if err := unstructured.SetNestedField(u.Object, numReplicas, "spec", "replicas"); err != nil {
			return err
		}
	}

	// Get the existing containers
	oldContainers, found, err := unstructured.NestedSlice(current.Object, "spec", "template", "spec",
		"containers")
	if err != nil || !found {
		return err
	}

	// Get the new containers
	containers, found, err := unstructured.NestedSlice(u.Object, "spec", "template", "spec",
		"containers")
	if err != nil || !found {
		return err
	}
	for index := range containers {
		name, found, err := unstructured.NestedString(containers[index].(map[string]interface{}),
			"name")
		if err != nil || !found {
			return err
		}
		envVars, foundVal := nestedEnvVar(name, oldContainers, "env")
		if !foundVal {
			continue
		}
		if err := unstructured.SetNestedField(containers[index].(map[string]interface{}), envVars,
			"env"); err != nil {
			return err
		}
	}
	if err := unstructured.SetNestedField(u.Object, containers, "spec", "template",
		"spec", "containers"); err != nil {
		return err
	}
	return nil
}

func nestedInt64OrFloat64(obj map[string]interface{}, fields ...string) (int64, bool) {
	val, found, err := unstructured.NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return 0, found
	}

	intVal, ok := val.(int64)
	if ok {
		return intVal, true
	}

	floatVal, ok := val.(float64)
	if ok {
		return int64(floatVal), true
	}

	return 0, false
}

func nestedEnvVar(name string, oldContainers []interface{}, fields ...string) (interface{}, bool) {
	for _, oldContainer := range oldContainers {
		oldName, found, err := unstructured.NestedString(oldContainer.(map[string]interface{}), "name")
		if err != nil || !found {
			return nil, false
		}
		if oldName == name {
			val, found, err := unstructured.NestedFieldCopy(oldContainer.(map[string]interface{}), fields...)
			if err != nil || !found {
				return nil, false
			}
			return val, true
		}
	}
	return nil, false
}
