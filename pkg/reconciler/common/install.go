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
	"context"
	"fmt"
	"strings"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/pkg/logging"
)

var (
	role                      mf.Predicate = mf.Any(mf.ByKind("ClusterRole"), mf.ByKind("Role"))
	rolebinding               mf.Predicate = mf.Any(mf.ByKind("ClusterRoleBinding"), mf.ByKind("RoleBinding"))
	webhook                   mf.Predicate = mf.Any(mf.ByKind("MutatingWebhookConfiguration"), mf.ByKind("ValidatingWebhookConfiguration"))
	webhookDependentResources mf.Predicate = byGV(schema.GroupKind{Group: "networking.internal.knative.dev", Kind: "Certificate"})
	gatewayNotMatch                        = "no matches for kind \"Gateway\""
)

// Install applies the manifest resources for the given version and updates the given
// status accordingly.
func Install(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Installing manifest")
	status := instance.GetStatus()
	// The Operator needs a higher level of permissions if it 'bind's non-existent roles.
	// To avoid this, we strictly order the manifest application as (Cluster)Roles, then
	// (Cluster)RoleBindings, then the rest of the manifest.
	if err := manifest.Filter(role).Apply(); err != nil {
		status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply (cluster)roles: %w", err)
	}
	if err := manifest.Filter(rolebinding).Apply(); err != nil {
		status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply (cluster)rolebindings: %w", err)
	}
	// Webhook configs are placeholder to be reconciled and configured by webhook controllers, not fully operational yet.
	// They are owned by SYSTEM_NAMESPACE and reconciled once webhook deployment is started.
	// In some cases a webhook config might be left on the cluster from previous uncompleted install/reinstall attempts.
	// Such pre-existing webhook config can block creation of ConfigMap and other resource that are validated, resulting
	// in a deadlock installation loop. For this reasons
	if err := InstallWebhookConfigs(ctx, manifest, instance); err != nil {
		return err
	}
	if err := manifest.Filter(mf.Not(mf.Any(role, rolebinding, webhook, webhookDependentResources))).Apply(); err != nil {
		status.MarkInstallFailed(err.Error())
		if ks, ok := instance.(*v1beta1.KnativeServing); ok && strings.Contains(err.Error(), gatewayNotMatch) &&
			(ks.Spec.Ingress == nil || ks.Spec.Ingress.Istio.Enabled) {
			errMessage := fmt.Errorf("please install istio or disable the istio ingress plugin: %w", err)
			status.MarkInstallFailed(errMessage.Error())
			return errMessage
		}

		return fmt.Errorf("failed to apply non rbac manifest: %w", err)
	}
	return nil
}

// InstallWebhookConfigs applies the Webhook manifest resources and updates the given status accordingly.
func InstallWebhookConfigs(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	logging.FromContext(ctx).Debug("Installing webhook configurations")
	status := instance.GetStatus()
	if err := manifest.Filter(webhook).Apply(); err != nil {
		status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply webhooks: %w", err)
	}
	return nil
}

// InstallWebhookDependentResources applies the Webhook dependent resources updates the given status accordingly.
func InstallWebhookDependentResources(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	status := instance.GetStatus()
	if err := manifest.Filter(webhookDependentResources).Apply(); err != nil {
		status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply webhooks: %w", err)
	}
	return nil
}

// MarkStatusSuccess updates the component status to success.
func MarkStatusSuccess(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	status := instance.GetStatus()
	status.MarkInstallSucceeded()
	status.SetVersion(TargetVersion(instance))
	return nil
}

// Uninstall removes all resources except CRDs, which are never deleted automatically.
func Uninstall(manifest *mf.Manifest) error {
	if err := manifest.Filter(mf.NoCRDs, mf.Not(mf.Any(role, rolebinding))).Delete(mf.IgnoreNotFound(true)); err != nil {
		return fmt.Errorf("failed to remove non-crd/non-rbac resources: %w", err)
	}
	// Delete Roles last, as they may be useful for human operators to clean up.
	if err := manifest.Filter(mf.Any(role, rolebinding)).Delete(mf.IgnoreNotFound(true)); err != nil {
		return fmt.Errorf("failed to remove rbac: %w", err)
	}
	return nil
}

func byGV(gk schema.GroupKind) mf.Predicate {
	return func(u *unstructured.Unstructured) bool {
		return u.GroupVersionKind().GroupKind() == gk
	}
}
