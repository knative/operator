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

package resources

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
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
	ks, err := clients.KnativeServing().Get(context.TODO(), names.KnativeServing, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("The operator does not have an existing KS operator CR: %s", names.KnativeServing)
	}
	// Add config to its spec
	ks.Spec = getTestKSOperatorCRSpec()

	// verify the default config map
	verifyDefaultConfig(t, ks, defaultsConfigMapName, clients, names)

	// verify the logging config map
	verifyLoggingConfig(t, loggingConfigMapName, clients, names)

	// Delete a single key/value pair
	verifySingleKeyDeletion(t, LoggingConfigKey, loggingConfigMapName, clients, names)

	// Verify HA config
	VerifyHADeployments(t, clients, names)

	// Use an empty map as the value
	verifyEmptyKey(t, DefaultsConfigKey, defaultsConfigMapName, clients, names)

	// Now remove the config from the spec and update
	verifyEmptySpec(t, loggingConfigMapName, clients, names)
}

func verifyDefaultConfig(t *testing.T, ks *v1beta1.KnativeServing, defaultsConfigMapName string, clients *test.Clients, names test.ResourceNames) {
	_, err := clients.KnativeServing().Update(context.TODO(), ks, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}

	// Verify the relevant configmaps have been updated
	err = WaitForConfigMap(defaultsConfigMapName, clients.KubeClient, func(m map[string]string) bool {
		return m["revision-timeout-seconds"] == "200"
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", defaultsConfigMapName)
	}
}

func verifyLoggingConfig(t *testing.T, loggingConfigMapName string, clients *test.Clients, names test.ResourceNames) {
	err := WaitForConfigMap(loggingConfigMapName, clients.KubeClient, func(m map[string]string) bool {
		return m["loglevel.controller"] == "debug" && m["loglevel.autoscaler"] == "debug"
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", loggingConfigMapName)
	}
}

func verifySingleKeyDeletion(t *testing.T, loggingConfigKey string, loggingConfigMapName string, clients *test.Clients, names test.ResourceNames) {
	ks, err := clients.KnativeServing().Get(context.TODO(), names.KnativeServing, metav1.GetOptions{})
	if err != nil || ks.Spec.Config[loggingConfigKey]["loglevel.autoscaler"] == "" {
		t.Fatalf("Existing KS operator CR lacks proper key: %v", ks.Spec.Config)
	}

	delete(ks.Spec.Config[loggingConfigKey], "loglevel.autoscaler")
	_, err = clients.KnativeServing().Update(context.TODO(), ks, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}

	// Verify the relevant configmap has been updated
	err = WaitForConfigMap(loggingConfigMapName, clients.KubeClient, func(m map[string]string) bool {
		_, autoscalerKeyExists := m["loglevel.autoscaler"]
		// deleted key/value pair should be removed from the target config map
		return m["loglevel.controller"] == "debug" && !autoscalerKeyExists
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", loggingConfigMapName)
	}
}

func verifyEmptyKey(t *testing.T, defaultsConfigKey string, defaultsConfigMapName string, clients *test.Clients, names test.ResourceNames) {
	ks, err := clients.KnativeServing().Get(context.TODO(), names.KnativeServing, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Existing KS operator CR gone: %s", names.KnativeServing)
	}

	ks.Spec.Config[defaultsConfigKey] = map[string]string{}
	_, err = clients.KnativeServing().Update(context.TODO(), ks, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}

	// Verify the relevant configmap has been updated and does not contain any keys except "_example"
	err = WaitForConfigMap(defaultsConfigMapName, clients.KubeClient, func(m map[string]string) bool {
		_, exampleExists := m["_example"]
		return len(m) == 1 && exampleExists
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", defaultsConfigMapName)
	}
}

func verifyEmptySpec(t *testing.T, loggingConfigMapName string, clients *test.Clients, names test.ResourceNames) {
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		ks, errGet := clients.KnativeServing().Get(context.TODO(), names.KnativeServing, metav1.GetOptions{})
		if errGet != nil {
			t.Fatalf("Existing KS operator CR gone: %s", names.KnativeServing)
		}
		ks.Spec = v1beta1.KnativeServingSpec{}
		if _, errUpdate := clients.KnativeServing().Update(context.TODO(), ks, metav1.UpdateOptions{}); errUpdate != nil {
			return false, nil
		}
		return true, nil
	})

	if waitErr != nil {
		t.Fatalf("The operator failed to update the Knative Serving CR")
	}

	err := WaitForConfigMap(loggingConfigMapName, clients.KubeClient, func(m map[string]string) bool {
		_, exists := m["loglevel.controller"]
		return !exists
	})
	if err != nil {
		t.Fatalf("The operator failed to update %s configmap", loggingConfigMapName)
	}
}

// DeleteAndVerifyDeployments verify whether all the deployments for knative serving are able to recreate, when they are deleted.
func DeleteAndVerifyDeployments(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	dpList, err := clients.KubeClient.AppsV1().Deployments(names.Namespace).List(context.TODO(), metav1.ListOptions{})
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
	if err := clients.KubeClient.AppsV1().Deployments(deployment.Namespace).Delete(context.TODO(), deployment.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Failed to delete deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
	}

	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		dep, err := clients.KubeClient.AppsV1().Deployments(deployment.Namespace).Get(context.TODO(), deployment.Name, metav1.GetOptions{})
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
	if err := clients.KnativeServing().Delete(context.TODO(), names.KnativeServing, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("KnativeServing %q failed to delete: %v", names.KnativeServing, err)
	}
	err := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		_, err := clients.KnativeServing().Get(context.TODO(), names.KnativeServing, metav1.GetOptions{})
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
	for _, u := range m.Filter(mf.NoCRDs, mf.Not(mf.ByKind("Namespace"))).Resources() {
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
	servings, err := clients.KnativeServingAll().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(servings.Items) > 0 {
		return errors.New("Unable to verify cluster-scoped resources are deleted if any KnativeServing exists")
	}
	return nil
}

// AssertKnativeObsoleteResource verifies if all obsolete resources disappear in the cluster
func AssertKnativeObsoleteResource(t *testing.T, clients *test.Clients, namespace string, obsResources []unstructured.Unstructured) {
	if err := WaitForKnativeResourceState(clients, namespace, obsResources, t.Logf,
		IsKnativeObsoleteResourceGone); err != nil {
		t.Fatalf("Knative obsolete resources failed to be removed: %v", err)
	}
}

// AssertKnativeDeploymentStatus verifies if the Knative deployments reach the READY status.
func AssertKnativeDeploymentStatus(t *testing.T, clients *test.Clients, namespace string, version string, existingVersion string, expectedDeployments []string) {
	if err := WaitForKnativeDeploymentState(clients, namespace, version, existingVersion, expectedDeployments, t.Logf,
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

// VerifyHADeployments verify whether all the deployments has scaled up.
func VerifyHADeployments(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	err := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		ks, err := clients.KnativeServing().Get(context.TODO(), names.KnativeServing, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("KnativeServing %q failed to get: %v", names.KnativeServing, err)
		}
		var two int32 = 2
		ks.Spec.HighAvailability = &base.HighAvailability{Replicas: &two}
		_, err = clients.KnativeServing().Update(context.TODO(), ks, metav1.UpdateOptions{})
		if err != nil {
			t.Logf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
			t.Logf("Retrying...")
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		t.Fatal("Timed out updating the HA on KnativeServing", err)
	}

	err = wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		deployments, err := clients.KubeClient.AppsV1().Deployments(names.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Failed to get any deployment under the namespace %q: %v", names.Namespace, err)
		}
		if len(deployments.Items) == 0 {
			t.Fatalf("No deployment under the namespace %q was found", names.Namespace)
		}
		for _, deploy := range deployments.Items {
			if got, want := deploy.Status.Replicas, int32(2); got != want {
				t.Logf("deployment %q: Status.Replicas = %d, want: %d", deploy.Name, got, want)
				t.Logf("Retrying...")
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		t.Fatal("Timed out waiting on KnativeServing to update the deployments", err)
	}

	// Get KnativeServing CR again to avoid "the object has been modified" error.
	ks, err := clients.KnativeServing().Get(context.TODO(), names.KnativeServing, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("KnativeServing %q failed to get: %v", names.KnativeServing, err)
	}
	ks.Spec.HighAvailability = nil
	_, err = clients.KnativeServing().Update(context.TODO(), ks, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}
	err = wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		deployments, err := clients.KubeClient.AppsV1().Deployments(names.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Failed to get any deployment under the namespace %q: %v", names.Namespace, err)
		}
		if len(deployments.Items) == 0 {
			t.Fatalf("No deployment under the namespace %q was found", names.Namespace)
		}
		for _, deploy := range deployments.Items {
			if got, want := deploy.Status.Replicas, int32(2); got != want {
				t.Logf("deployment %q: Status.Replicas = %d, want: %d", deploy.Name, got, want)
				t.Logf("Retrying...")
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		t.Fatal("Timed out waiting on KnativeServing to delete", err)
	}
}

// DeleteAndVerifyEventingDeployments verify whether all the deployments for knative eventing are able to recreate, when they are deleted.
func DeleteAndVerifyEventingDeployments(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	dpList, err := clients.KubeClient.AppsV1().Deployments(names.Namespace).List(context.TODO(), metav1.ListOptions{})
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
	if err := clients.KubeClient.AppsV1().Deployments(deployment.Namespace).Delete(context.TODO(), deployment.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Failed to delete deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
	}

	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		dep, err := clients.KubeClient.AppsV1().Deployments(deployment.Namespace).Get(context.TODO(), deployment.Name, metav1.GetOptions{})
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
	if err := clients.KnativeEventing().Delete(context.TODO(), names.KnativeEventing, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("KnativeEventing %q failed to delete: %v", names.KnativeEventing, err)
	}
	err := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		_, err := clients.KnativeEventing().Get(context.TODO(), names.KnativeEventing, metav1.GetOptions{})
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
	for _, u := range m.Filter(mf.NoCRDs, mf.Not(mf.ByKind("Namespace"))).Resources() {
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
	eventings, err := clients.KnativeEventingAll().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(eventings.Items) > 0 {
		return errors.New("Unable to verify cluster-scoped resources are deleted if any KnativeEventing exists")
	}
	return nil
}
