/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/operator/test"
)

// AssertKSOperatorCRReadyStatus verifies if the KnativeServing reaches the READY status.
func AssertKSOperatorCRReadyStatus(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	if _, err := WaitForKnativeServingState(clients.KnativeServing(), names.KnativeServing,
		IsKnativeServingReady); err != nil {
		t.Fatalf("KnativeService %q failed to get to the READY status: %v", names.KnativeServing, err)
	}
}

// KSOperatorCRVerifyConfiguration verifies that KnativeServing config is set properly
func KSOperatorCRVerifyConfiguration(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	// We'll arbitrarily choose logging and defaults config
	loggingConfigMapName := fmt.Sprintf("%s/config-%s", names.Namespace, LoggingConfigKey)
	defaultsConfigMapName := fmt.Sprintf("%s/config-%s", names.Namespace, DefaultsConfigKey)
	// Get the existing KS without any spec
	ks, err := clients.KnativeServing().Get(names.KnativeServing, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("The operator does not have an existing KS operator CR: %s", names.KnativeServing)
	}
	// Add config to its spec
	ks.Spec = getTestKSOperatorCRSpec()

	// verify the default config map
	ks = verifyDefaultConfig(t, ks, defaultsConfigMapName, clients, names)

	// verify the logging config map
	verifyLoggingConfig(t, ks, loggingConfigMapName, clients, names)

	// Delete a single key/value pair
	ks = verifySingleKeyDeletion(t, ks, LoggingConfigKey, loggingConfigMapName, clients, names)

	// Use an empty map as the value
	ks = verifyEmptyKey(t, ks, DefaultsConfigKey, defaultsConfigMapName, clients, names)

	// Now remove the config from the spec and update
	verifyEmptySpec(t, ks, loggingConfigMapName, clients, names)
}

func verifyDefaultConfig(t *testing.T, ks *v1alpha1.KnativeServing, defaultsConfigMapName string, clients *test.Clients,
	names test.ResourceNames) *v1alpha1.KnativeServing {
	ks, err := clients.KnativeServing().Update(ks)
	if err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}
	// Verify the relevant configmaps have been updated
	err = WaitForConfigMap(defaultsConfigMapName, clients.KubeClient.Kube, func(m map[string]string) bool {
		return m["revision-timeout-seconds"] == "200"
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", defaultsConfigMapName)
	}
	return ks
}

func verifyLoggingConfig(t *testing.T, ks *v1alpha1.KnativeServing, loggingConfigMapName string, clients *test.Clients,
	names test.ResourceNames) {
	err := WaitForConfigMap(loggingConfigMapName, clients.KubeClient.Kube, func(m map[string]string) bool {
		return m["loglevel.controller"] == "debug" && m["loglevel.autoscaler"] == "debug"
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", loggingConfigMapName)
	}
}

func verifySingleKeyDeletion(t *testing.T, ks *v1alpha1.KnativeServing, loggingConfigKey string,
	loggingConfigMapName string, clients *test.Clients, names test.ResourceNames) *v1alpha1.KnativeServing {
	delete(ks.Spec.Config[loggingConfigKey], "loglevel.autoscaler")
	ks, err := clients.KnativeServing().Update(ks)
	if err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}
	// Verify the relevant configmap has been updated
	err = WaitForConfigMap(loggingConfigMapName, clients.KubeClient.Kube, func(m map[string]string) bool {
		_, autoscalerKeyExists := m["loglevel.autoscaler"]
		// deleted key/value pair should be removed from the target config map
		return m["loglevel.controller"] == "debug" && !autoscalerKeyExists
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", loggingConfigMapName)
	}
	return ks
}

func verifyEmptyKey(t *testing.T, ks *v1alpha1.KnativeServing, defaultsConfigKey string,
	defaultsConfigMapName string, clients *test.Clients, names test.ResourceNames) *v1alpha1.KnativeServing {
	ks.Spec.Config[defaultsConfigKey] = map[string]string{}
	ks, err := clients.KnativeServing().Update(ks)
	if err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}
	// Verify the relevant configmap has been updated and does not contain any keys except "_example"
	err = WaitForConfigMap(defaultsConfigMapName, clients.KubeClient.Kube, func(m map[string]string) bool {
		_, exampleExists := m["_example"]
		return len(m) == 1 && exampleExists
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", defaultsConfigMapName)
	}
	return ks
}

func verifyEmptySpec(t *testing.T, ks *v1alpha1.KnativeServing, loggingConfigMapName string, clients *test.Clients,
	names test.ResourceNames) {
	ks.Spec = v1alpha1.KnativeServingSpec{}
	if _, err := clients.KnativeServing().Update(ks); err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}
	err := WaitForConfigMap(loggingConfigMapName, clients.KubeClient.Kube, func(m map[string]string) bool {
		_, exists := m["loglevel.controller"]
		return !exists
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", loggingConfigMapName)
	}
}

// DeleteAndVerifyDeployments verify whether all the deployments for knative serving are able to recreate, when they are deleted.
func DeleteAndVerifyDeployments(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	dpList, err := clients.KubeClient.Kube.AppsV1().Deployments(names.Namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to get any deployment under the namespace %q: %v",
			test.ServingOperatorNamespace, err)
	}
	if len(dpList.Items) == 0 {
		t.Fatalf("No deployment under the namespace %q was found",
			test.ServingOperatorNamespace)
	}
	// Delete the first deployment and verify the operator recreates it
	deployment := dpList.Items[0]
	if err := clients.KubeClient.Kube.AppsV1().Deployments(deployment.Namespace).Delete(deployment.Name,
		&metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Failed to delete deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
	}

	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		dep, err := clients.KubeClient.Kube.AppsV1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			// If the deployment is not found, we continue to wait for the availability.
			if apierrs.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return IsDeploymentAvailable(dep)
	})

	if waitErr != nil {
		t.Fatalf("The deployment %s/%s failed to reach the desired state: %v", deployment.Namespace, deployment.Name, err)
	}

	if _, err := WaitForKnativeServingState(clients.KnativeServing(), test.OperatorName,
		IsKnativeServingReady); err != nil {
		t.Fatalf("KnativeService %q failed to reach the desired state: %v", test.OperatorName, err)
	}
	t.Logf("The deployment %s/%s reached the desired state.", deployment.Namespace, deployment.Name)
}

// KSOperatorCRDelete deletes tha KnativeServing to see if all resources will be deleted
func KSOperatorCRDelete(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	if err := clients.KnativeServing().Delete(names.KnativeServing, &metav1.DeleteOptions{}); err != nil {
		t.Fatalf("KnativeServing %q failed to delete: %v", names.KnativeServing, err)
	}
	err := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		_, err := clients.KnativeServing().Get(names.KnativeServing, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		t.Fatal("Timed out waiting on KnativeServing to delete", err)
	}
	_, b, _, _ := runtime.Caller(0)
	m, err := mfc.NewManifest(filepath.Join((filepath.Dir(b)+"/.."), "config/"), clients.Config)
	if err != nil {
		t.Fatal("Failed to load manifest", err)
	}
	if err := verifyNoKSOperatorCR(clients); err != nil {
		t.Fatal(err)
	}

	// verify all but the CRD's and the Namespace are gone
	for _, u := range m.Filter(mf.NoCRDs, mf.Not(mf.Any(mf.ByKind("Namespace")))).Resources() {
		if _, err := m.Client.Get(&u); !apierrs.IsNotFound(err) {
			t.Fatalf("The %s %s failed to be deleted: %v", u.GetKind(), u.GetName(), err)
		}
	}
	// verify all the CRD's remain
	for _, u := range m.Filter(mf.CRDs).Resources() {
		if _, err := m.Client.Get(&u); apierrs.IsNotFound(err) {
			t.Fatalf("The %s CRD was deleted", u.GetName())
		}
	}
}

func verifyNoKSOperatorCR(clients *test.Clients) error {
	servings, err := clients.KnativeServingAll().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(servings.Items) > 0 {
		return errors.New("Unable to verify cluster-scoped resources are deleted if any KnativeServing exists")
	}
	return nil
}

// AssertKnativeDeploymentStatus verifies if the Knative deployments reach the READY status.
func AssertKnativeDeploymentStatus(t *testing.T, clients *test.Clients, namespace string, expectedDeployments []string) {
	if err := WaitForKnativeDeploymentState(clients, namespace, expectedDeployments, t.Logf,
		IsKnativeDeploymentReady); err != nil {
		t.Fatalf("Knative Serving deployments failed to meet the expected deployments: %v", err)
	}
}

// AssertKEOperatorCRReadyStatus verifies if the KnativeEventing can reach the READY status.
func AssertKEOperatorCRReadyStatus(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	if _, err := WaitForKnativeEventingState(clients.KnativeEventing(), names.KnativeEventing,
		IsKnativeEventingReady); err != nil {
		t.Fatalf("KnativeService %q failed to get to the READY status: %v", names.KnativeEventing, err)
	}
}

// DeleteAndVerifyEventingDeployments verify whether all the deployments for knative eventing are able to recreate, when they are deleted.
func DeleteAndVerifyEventingDeployments(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	dpList, err := clients.KubeClient.Kube.AppsV1().Deployments(names.Namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to get any deployment under the namespace %q: %v",
			test.EventingOperatorNamespace, err)
	}
	if len(dpList.Items) == 0 {
		t.Fatalf("No deployment under the namespace %q was found",
			test.EventingOperatorNamespace)
	}
	// Delete the first deployment and verify the operator recreates it
	deployment := dpList.Items[0]
	if err := clients.KubeClient.Kube.AppsV1().Deployments(deployment.Namespace).Delete(deployment.Name,
		&metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Failed to delete deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
	}

	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		dep, err := clients.KubeClient.Kube.AppsV1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			// If the deployment is not found, we continue to wait for the availability.
			if apierrs.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return IsDeploymentAvailable(dep)
	})

	if waitErr != nil {
		t.Fatalf("The deployment %s/%s failed to reach the desired state: %v", deployment.Namespace, deployment.Name, err)
	}

	if _, err := WaitForKnativeEventingState(clients.KnativeEventing(), test.OperatorName,
		IsKnativeEventingReady); err != nil {
		t.Fatalf("KnativeService %q failed to reach the desired state: %v", test.OperatorName, err)
	}
	t.Logf("The deployment %s/%s reached the desired state.", deployment.Namespace, deployment.Name)
}

// KEOperatorCRDelete deletes tha KnativeEventing to see if all resources will be deleted
func KEOperatorCRDelete(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	if err := clients.KnativeEventing().Delete(names.KnativeEventing, &metav1.DeleteOptions{}); err != nil {
		t.Fatalf("KnativeEventing %q failed to delete: %v", names.KnativeEventing, err)
	}
	err := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		_, err := clients.KnativeEventing().Get(names.KnativeEventing, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		t.Fatal("Timed out waiting on KnativeServing to delete", err)
	}
	_, b, _, _ := runtime.Caller(0)
	m, err := mfc.NewManifest(filepath.Join((filepath.Dir(b)+"/.."), "config/"), clients.Config)
	if err != nil {
		t.Fatal("Failed to load manifest", err)
	}
	if err := verifyNoKnativeEventings(clients); err != nil {
		t.Fatal(err)
	}
	// verify all but the CRD's and the Namespace are gone
	for _, u := range m.Filter(mf.NoCRDs, mf.Not(mf.Any(mf.ByKind("Namespace")))).Resources() {
		if _, err := m.Client.Get(&u); !apierrs.IsNotFound(err) {
			t.Fatalf("The %s %s failed to be deleted: %v", u.GetKind(), u.GetName(), err)
		}
	}
	// verify all the CRD's remain
	for _, u := range m.Filter(mf.CRDs).Resources() {
		if _, err := m.Client.Get(&u); apierrs.IsNotFound(err) {
			t.Fatalf("The %s CRD was deleted", u.GetName())
		}
	}
}

func verifyNoKnativeEventings(clients *test.Clients) error {
	eventings, err := clients.KnativeEventingAll().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(eventings.Items) > 0 {
		return errors.New("Unable to verify cluster-scoped resources are deleted if any KnativeEventing exists")
	}
	return nil
}
