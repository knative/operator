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

// knativeserving.go provides methods to perform actions on the KnativeEventing resource.

package resources

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	eventingv1alpha1 "knative.dev/operator/pkg/client/clientset/versioned/typed/operator/v1alpha1"
	"knative.dev/operator/test"
	"knative.dev/pkg/test/logging"
)

// WaitForKnativeEventingState polls the status of the KnativeEventing called name
// from client every `interval` until `inState` returns `true` indicating it
// is done, returns an error or timeout.
func WaitForKnativeEventingState(clients eventingv1alpha1.KnativeEventingInterface, name string,
	inState func(s *v1alpha1.KnativeEventing, err error) (bool, error)) (*v1alpha1.KnativeEventing, error) {
	span := logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForKnativeEventingState/%s/%s", name, "KnativeEventingIsReady"))
	defer span.End()

	var lastState *v1alpha1.KnativeEventing
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err := clients.Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "KnativeEventing %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

// EnsureKnativeEventingExists creates a KnativeEventingServing with the name names.KnativeEventing under the namespace names.Namespace.
func EnsureKnativeEventingExists(clients eventingv1alpha1.KnativeEventingInterface, names test.ResourceNames) (*v1alpha1.KnativeEventing, error) {
	// If this function is called by the upgrade tests, we only create the custom resource, if it does not exist.
	ke, err := clients.Get(names.KnativeEventing, metav1.GetOptions{})
	if apierrs.IsNotFound(err) {
		ke := &v1alpha1.KnativeEventing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      names.KnativeEventing,
				Namespace: names.Namespace,
			},
		}
		return clients.Create(ke)
	}
	return ke, err
}

// IsKnativeEventingReady will check the status conditions of the KnativeEventing and return true if the KnativeEventing is ready.
func IsKnativeEventingReady(s *v1alpha1.KnativeEventing, err error) (bool, error) {
	return s.Status.IsReady(), err
}

// WaitForKnativeEventingDeploymentState polls the status of the Knative deployments every `interval`
// until `inState` returns `true` indicating the deployments match the desired deployments.
func WaitForKnativeEventingDeploymentState(clients *test.Clients, namespace string, expectedDeployments []string, logf logging.FormatLogger,
	inState func(deps *v1.DeploymentList, expectedDeployments []string, err error, logf logging.FormatLogger) (bool, error)) error {
	span := logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForKnativeDeploymentState/%s/%s", expectedDeployments, "KnativeDeploymentIsReady"))
	defer span.End()

	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		dpList, err := clients.KubeClient.Kube.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
		return inState(dpList, expectedDeployments, err, logf)
	})

	return waitErr
}

// IsKnativeEventingDeploymentReady will check the status conditions of the deployments and return true if the deployments meet the desired status.
func IsKnativeEventingDeploymentReady(dpList *v1.DeploymentList, expectedDeployments []string, err error,
	logf logging.FormatLogger) (bool, error) {
	if err != nil {
		return false, err
	}
	if len(dpList.Items) != len(expectedDeployments) {
		logf("The expected number of deployments is %v, and got %v.", len(expectedDeployments), len(dpList.Items))
		return false, nil
	}
	for _, deployment := range dpList.Items {
		if !stringInList(deployment.Name, expectedDeployments) {
			logf("The deployment %v is not found in the expected list of deployment.", deployment.Name)
			return false, nil
		}
		for _, c := range deployment.Status.Conditions {
			if c.Type == v1.DeploymentAvailable && c.Status != corev1.ConditionTrue {
				logf("The deployment %v is not ready.", deployment.Name)
				return false, nil
			}
		}
	}
	return true, nil
}
