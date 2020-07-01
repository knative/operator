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
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"knative.dev/pkg/apis"

	apierrs "k8s.io/apimachinery/pkg/api/errors"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/reconciler/common"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
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

	for _, deploymentName := range expectedDeployments {
		dep := deploymentInList(deploymentName, dpList.Items)
		if dep == nil {
			logf("The deployment %v is not found.", deploymentName)
			return false, nil
		}
		for _, c := range dep.Status.Conditions {
			if c.Type == v1.DeploymentAvailable && c.Status != corev1.ConditionTrue {
				logf("The deployment %v is not ready.", dep.Name)
				return false, nil
			}
		}
	}

	return true, nil
}

// GetExpectedDeployments will return an array of deployment resources based on the version for the knative
// component.
func GetExpectedDeployments(t *testing.T, instance v1alpha1.KComponent) (mf.Manifest, []string) {
	manifest, err := common.InstalledManifest(instance)
	if err != nil {
		t.Fatalf("Failed to get the manifest for Knative: %v", err)
	}

	deployments := []string{}
	for _, resource := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		deployments = append(deployments, resource.GetName())
	}
	return manifest, removeDuplications(deployments)
}

// SetKodataDir will set the env var KO_DATA_PATH into the path of the kodata of this repository.
func SetKodataDir() {
	_, b, _, _ := runtime.Caller(0)
	koPath := filepath.Join(getParentDir(b, 2), "cmd/operator/kodata")
	os.Setenv(common.KoEnvKey, koPath)
}

func getParentDir(path string, times int) string {
	if times < 0 {
		return path
	}

	if times == 0 {
		return filepath.Dir(path)
	}

	return getParentDir(filepath.Dir(path), times-1)
}

func removeDuplications(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// WaitForKnativeResourceState returns the status of whether all obsolete resources are removed
func WaitForKnativeResourceState(clients *test.Clients, namespace string,
	obsResources []unstructured.Unstructured, logf logging.FormatLogger, inState func(clients *test.Clients,
		namespace string, obsResources []unstructured.Unstructured, logf logging.FormatLogger) (bool, error)) error {
	span := logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForKnativeResourceState/%s/%s", obsResources, "KnativeObsoleteResourceIsGone"))
	defer span.End()

	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		return inState(clients, namespace, obsResources, logf)
	})

	return waitErr
}

// IsKnativeObsoleteResourceGone check the status conditions of the resources and return true if the obsolete resources are removed.
func IsKnativeObsoleteResourceGone(clients *test.Clients, namespace string, obsResources []unstructured.Unstructured,
	logf logging.FormatLogger) (bool, error) {
	for _, resource := range obsResources {
		gvr := apis.KindToResource(resource.GroupVersionKind())
		var err error
		if resource.GetNamespace() != "" {
			// This is a namespaced resource
			_, err = clients.Dynamic.Resource(gvr).Namespace(namespace).Get(resource.GetName(), metav1.GetOptions{})
		} else {
			// This is a clustered resource
			_, err = clients.Dynamic.Resource(gvr).Get(resource.GetName(), metav1.GetOptions{})
		}
		if !apierrs.IsNotFound(err) {
			logf("The resource %v still exists.", resource.GetName())
			return false, nil
		}
	}
	return true, nil
}
