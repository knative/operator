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

package reconciler

import (
	"os"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/system"
)

func TestRemovePreUnifiedResources(t *testing.T) {
	tests := []struct {
		name    string
		in      []runtime.Object
		oldName string
	}{{
		name:    "none exist",
		oldName: "test1",
	}, {
		name: "delete deployment only",
		in: []runtime.Object{
			deployment("test", "test2"),
		},
		oldName: "test2",
	}, {
		name: "delete serviceaccount only",
		in: []runtime.Object{
			sa("test", "test3"),
		},
		oldName: "test3",
	}, {
		name: "delete both",
		in: []runtime.Object{
			deployment("test", "test4"),
			sa("test", "test4"),
		},
		oldName: "test4",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ns := "test"
			os.Setenv(system.NamespaceEnvKey, ns)
			fake := fakekube.NewSimpleClientset(test.in...)
			if err := RemovePreUnifiedResources(fake, test.oldName); err != nil {
				t.Fatalf("RemovePreUnifiedResources() = %v, want nil", err)
			}

			if _, err := fake.CoreV1().ServiceAccounts(ns).Get(test.oldName, metav1.GetOptions{}); !apierrs.IsNotFound(err) {
				t.Fatalf("ServiceAccount %s was still present: %v", test.oldName, err)
			}
			if _, err := fake.AppsV1().Deployments(ns).Get(test.oldName, metav1.GetOptions{}); !apierrs.IsNotFound(err) {
				t.Fatalf("Deployment %s was still present: %v", test.oldName, err)
			}
		})
	}
}

func deployment(ns, name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
}

func sa(ns, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
}
