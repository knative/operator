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

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/operator/test"
	"knative.dev/pkg/test/logging"
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

// WaitForKnativeDeploymentState polls the status of the Knative deployments every `interval`
// until `inState` returns `true` indicating the deployments match the desired deployments.
func WaitForKnativeDeploymentState(clients *test.Clients, namespace string, expectedDeployments []string, logf logging.FormatLogger,
	inState func(deps *v1.DeploymentList, expectedDeployments []string, err error, logf logging.FormatLogger) (bool, error)) error {
	span := logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForKnativeDeploymentState/%s/%s", expectedDeployments, "KnativeDeploymentIsReady"))
	defer span.End()

	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		dpList, err := clients.KubeClient.Kube.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
		return inState(dpList, expectedDeployments, err, logf)
	})

	return waitErr
}

// IsKnativeDeploymentReady will check the status conditions of the deployments and return true if the deployments meet the desired status.
func IsKnativeDeploymentReady(dpList *v1.DeploymentList, expectedDeployments []string, err error,
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
