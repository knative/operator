/*
Copyright 2022 The Knative Authors

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

package main

import (
	"context"
	"log"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	"knative.dev/operator/pkg/apis/operator"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/certificates"
	"knative.dev/pkg/webhook/resourcesemantics/conversion"
)

func main() {
	secretName := "operator-webhook-certs"
	context := signals.NewContext()

	// Set up a signal context with our webhook options
	ctx := webhook.WithOptions(context, webhook.Options{
		ServiceName: webhook.NameFromEnv(),
		Port:        webhook.PortFromEnv(8443),
		SecretName:  secretName,
	})

	cfg := injection.ParseAndGetRESTConfigOrDie()
	err := ensureWebhookSecretExists(context, cfg, secretName)
	if err != nil {
		log.Fatal("Error there is an issue with the Secret called "+secretName+" for the operator webhook: ", zap.Error(err))
	}

	sharedmain.MainWithConfig(ctx, webhook.NameFromEnv(), cfg,
		certificates.NewController,
		newConversionController)
}

func newConversionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	var (
		v1beta1  = operatorv1beta1.SchemeGroupVersion.Version
		v1alpha1 = operatorv1alpha1.SchemeGroupVersion.Version
	)

	return conversion.NewConversionController(ctx,
		// The path on which to serve the webhook
		"/resource-conversion",

		// Specify the types of custom resource definitions that should be converted
		map[schema.GroupKind]conversion.GroupKindConversion{
			operatorv1beta1.Kind("KnativeServing"): {
				DefinitionName: operator.KnativeServingResource.String(),
				HubVersion:     v1alpha1,
				Zygotes: map[string]conversion.ConvertibleObject{
					v1alpha1: &operatorv1alpha1.KnativeServing{},
					v1beta1:  &operatorv1beta1.KnativeServing{},
				},
			},
			operatorv1beta1.Kind("KnativeEventing"): {
				DefinitionName: operator.KnativeEventingResource.String(),
				HubVersion:     v1alpha1,
				Zygotes: map[string]conversion.ConvertibleObject{
					v1alpha1: &operatorv1alpha1.KnativeEventing{},
					v1beta1:  &operatorv1beta1.KnativeEventing{},
				},
			},
		},

		// A function that infuses the context passed to ConvertTo/ConvertFrom/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return ctx
		},
	)
}

func ensureWebhookSecretExists(context context.Context, cfg *rest.Config, name string) error {
	ctx, _ := injection.Default.SetupInformers(context, cfg)
	_, err := kubeclient.Get(ctx).CoreV1().Secrets(system.Namespace()).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Create the secret
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      name,
				Labels: map[string]string{"app.kubernetes.io/component": "webhook",
					"app.kubernetes.io/part-of": "knative-operator"},
			},
			Type: v1.SecretTypeOpaque,
		}
		_, err = kubeclient.Get(ctx).CoreV1().Secrets(system.Namespace()).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
