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
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"knative.dev/pkg/test/logging"
	"knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	servingv1alpha1 "knative.dev/serving-operator/pkg/client/clientset/versioned/typed/serving/v1alpha1"
	"knative.dev/serving-operator/test"
)

const (
	// Interval specifies the time between two polls.
	Interval = 10 * time.Second
	// Timeout specifies the timeout for the function PollImmediate to reach a certain status.
	Timeout = 5 * time.Minute
	// LoggingConfigKey specifies specifies the key name of the logging config map.
	LoggingConfigKey = "logging"
	// DefaultsConfigKey specifies the key name of the default config map.
	DefaultsConfigKey = "defaults"
)

// WaitForKnativeServingState polls the status of the KnativeServing called name
// from client every `interval` until `inState` returns `true` indicating it
// is done, returns an error or timeout.
func WaitForKnativeServingState(clients servingv1alpha1.KnativeServingInterface, name string,
	inState func(s *v1alpha1.KnativeServing, err error) (bool, error)) (*v1alpha1.KnativeServing, error) {
	span := logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForKnativeServingState/%s/%s", name, "KnativeServingIsReady"))
	defer span.End()

	var lastState *v1alpha1.KnativeServing
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err := clients.Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "knativeserving %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

// EnsureKnativeServingExists creates a KnativeServing with the name names.KnativeServing under the namespace names.Namespace, if it does not exist.
func EnsureKnativeServingExists(clients servingv1alpha1.KnativeServingInterface, names test.ResourceNames) (*v1alpha1.KnativeServing, error) {
	// If this function is called by the upgrade tests, we only create the custom resource, if it does not exist.
	ks, err := clients.Get(names.KnativeServing, metav1.GetOptions{})
	if apierrs.IsNotFound(err) {
		ks := &v1alpha1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      names.KnativeServing,
				Namespace: names.Namespace,
			},
		}
		return clients.Create(ks)
	}
	return ks, err
}

// WaitForConfigMap takes a condition function that evaluates ConfigMap data
func WaitForConfigMap(name string, client *kubernetes.Clientset, fn func(map[string]string) bool) error {
	ns, cm, _ := cache.SplitMetaNamespaceKey(name)
	return wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		cm, err := client.CoreV1().ConfigMaps(ns).Get(cm, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return fn(cm.Data), nil
	})
}

// IsKnativeServingReady will check the status conditions of the KnativeServing and return true if the KnativeServing is ready.
func IsKnativeServingReady(s *v1alpha1.KnativeServing, err error) (bool, error) {
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

func getTestKSOperatorCRSpec() v1alpha1.KnativeServingSpec {
	return v1alpha1.KnativeServingSpec{
		Config: map[string]map[string]string{
			DefaultsConfigKey: {
				"revision-timeout-seconds": "200",
			},
			LoggingConfigKey: {
				"loglevel.controller": "debug",
				"loglevel.autoscaler": "debug",
			},
		},
	}
}

// WaitForKnativeServingDeploymentState polls the status of the Knative deployments every `interval`
// until `inState` returns `true` indicating the deployments match the desired deployments.
func WaitForKnativeServingDeploymentState(clients *test.Clients, namespace string, expectedDeployments []string,
	inState func(deps *v1.DeploymentList, expectedDeployments []string, err error) (bool, error)) (*v1alpha1.KnativeServing, error) {
	span := logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForKnativeDeploymentState/%s/%s", expectedDeployments, "KnativeDeploymentIsReady"))
	defer span.End()

	var lastState *v1alpha1.KnativeServing
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		dpList, err := clients.KubeClient.Kube.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
		return inState(dpList, expectedDeployments, err)
	})

	if waitErr != nil {
		return lastState, waitErr
	}
	return lastState, nil
}

// IsKnativeServingDeploymentReady will check the status conditions of the deployments and return true if the deployments meet the desired status.
func IsKnativeServingDeploymentReady(dpList *v1.DeploymentList, expectedDeployments []string, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	if len(dpList.Items) != len(expectedDeployments) {
		errMessage := fmt.Sprintf("The expected number of deployments is %v, and got %v.", len(expectedDeployments), len(dpList.Items))
		return false, errors.New(errMessage)
	}
	for _, deployment := range dpList.Items {
		if !stringInList(deployment.Name, expectedDeployments) {
			errMessage := fmt.Sprintf("The deployment %v is not found in the expected list of deployment.", deployment.Name)
			return false, errors.New(errMessage)
		}
		for _, c := range deployment.Status.Conditions {
			if c.Type == v1.DeploymentAvailable && c.Status != corev1.ConditionTrue {
				errMessage := fmt.Sprintf("The deployment %v is not ready.", deployment.Name)
				return false, errors.New(errMessage)
			}
		}
	}
	return true, nil
}

func stringInList(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
