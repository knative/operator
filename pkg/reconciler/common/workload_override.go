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
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/base"
)

// resourceHandler defines the interface for handling different Kubernetes resource types
type resourceHandler interface {
	GetPodTemplateSpec() *corev1.PodTemplateSpec
	SetReplicas(*int32)
	GetObject() metav1.Object
}

// deploymentHandler implements resourceHandler for Deployments
type deploymentHandler struct {
	deployment *appsv1.Deployment
}

func (d *deploymentHandler) GetPodTemplateSpec() *corev1.PodTemplateSpec {
	return &d.deployment.Spec.Template
}

func (d *deploymentHandler) SetReplicas(replicas *int32) {
	d.deployment.Spec.Replicas = replicas
}

func (d *deploymentHandler) GetObject() metav1.Object {
	return d.deployment
}

// statefulSetHandler implements resourceHandler for StatefulSets
type statefulSetHandler struct {
	statefulSet *appsv1.StatefulSet
}

func (s *statefulSetHandler) GetPodTemplateSpec() *corev1.PodTemplateSpec {
	return &s.statefulSet.Spec.Template
}

func (s *statefulSetHandler) SetReplicas(replicas *int32) {
	s.statefulSet.Spec.Replicas = replicas
}

func (s *statefulSetHandler) GetObject() metav1.Object {
	return s.statefulSet
}

// jobHandler implements resourceHandler for Jobs
type jobHandler struct {
	job *batchv1.Job
}

func (j *jobHandler) GetPodTemplateSpec() *corev1.PodTemplateSpec {
	return &j.job.Spec.Template
}

func (j *jobHandler) SetReplicas(replicas *int32) {
	// Jobs don't have replicas, so this is a no-op
}

func (j *jobHandler) GetObject() metav1.Object {
	return j.job
}

// createResourceHandler creates the appropriate resource handler based on the resource type
func createResourceHandler(u *unstructured.Unstructured) (resourceHandler, error) {
	switch u.GetKind() {
	case "Deployment":
		deployment := &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
			return nil, err
		}
		return &deploymentHandler{deployment: deployment}, nil
	case "StatefulSet":
		ss := &appsv1.StatefulSet{}
		if err := scheme.Scheme.Convert(u, ss, nil); err != nil {
			return nil, err
		}
		return &statefulSetHandler{statefulSet: ss}, nil
	case "Job":
		job := &batchv1.Job{}
		if err := scheme.Scheme.Convert(u, job, nil); err != nil {
			return nil, err
		}
		return &jobHandler{job: job}, nil
	default:
		return nil, nil
	}
}

// OverridesTransform transforms deployments based on the configuration in `spec.overrides`.
func OverridesTransform(overrides []base.WorkloadOverride, log *zap.SugaredLogger) mf.Transformer {
	if overrides == nil {
		return nil
	}
	return func(u *unstructured.Unstructured) error {
		for _, override := range overrides {
			// Handle HPA separately since it doesn't follow the resource handler pattern
			if u.GetKind() == "HorizontalPodAutoscaler" && override.Replicas != nil && u.GetName() == getHPAName(override.Name) {
				overrideReplicas := int64(*override.Replicas)
				if err := hpaTransform(u, overrideReplicas); err != nil {
					return err
				}
				continue
			}

			// Check if this resource matches the override
			var matches bool
			switch u.GetKind() {
			case "Deployment", "StatefulSet":
				matches = u.GetName() == override.Name
			case "Job":
				matches = u.GetGenerateName() == override.Name
			default:
				continue
			}

			if !matches {
				continue
			}

			// Create the appropriate resource handler
			handler, err := createResourceHandler(u)
			if err != nil {
				return err
			}
			if handler == nil {
				continue
			}

			// Apply replicas if specified and not controlled by HPA
			if override.Replicas != nil && !hasHorizontalPodOrCustomAutoscaler(override.Name) {
				handler.SetReplicas(override.Replicas)
			}

			// Apply all other overrides
			obj := handler.GetObject()
			ps := handler.GetPodTemplateSpec()

			replaceLabels(&override, obj, ps)
			replaceAnnotations(&override, obj, ps)
			replaceNodeSelector(&override, ps)
			replaceTopologySpreadConstraints(&override, ps)
			replaceTolerations(&override, ps)
			replaceAffinities(&override, ps)
			replaceResources(&override, ps)
			replaceEnv(&override, ps)
			replaceProbes(&override, ps)
			replaceHostNetwork(&override, ps)

			// Convert back to unstructured
			if err := scheme.Scheme.Convert(obj, u, nil); err != nil {
				return err
			}

			// Avoid superfluous updates from converted zero defaults
			u.SetCreationTimestamp(metav1.Time{})
		}
		return nil
	}
}

func replaceAnnotations(override *base.WorkloadOverride, obj metav1.Object, ps *corev1.PodTemplateSpec) {
	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(map[string]string{})
	}
	if ps.GetAnnotations() == nil {
		ps.SetAnnotations(map[string]string{})
	}
	for key, val := range override.Annotations {
		obj.GetAnnotations()[key] = val
		ps.Annotations[key] = val
	}
}

func replaceLabels(override *base.WorkloadOverride, obj metav1.Object, ps *corev1.PodTemplateSpec) {
	if obj.GetLabels() == nil {
		obj.SetLabels(map[string]string{})
	}
	if ps.GetLabels() == nil {
		ps.Labels = map[string]string{}
	}
	for key, val := range override.Labels {
		obj.GetLabels()[key] = val
		ps.Labels[key] = val
	}
}

func replaceNodeSelector(override *base.WorkloadOverride, ps *corev1.PodTemplateSpec) {
	if len(override.NodeSelector) > 0 {
		ps.Spec.NodeSelector = override.NodeSelector
	}
}

func replaceTopologySpreadConstraints(override *base.WorkloadOverride, ps *corev1.PodTemplateSpec) {
	if len(override.TopologySpreadConstraints) > 0 {
		ps.Spec.TopologySpreadConstraints = override.TopologySpreadConstraints
	}
}

func replaceTolerations(override *base.WorkloadOverride, ps *corev1.PodTemplateSpec) {
	if len(override.Tolerations) > 0 {
		ps.Spec.Tolerations = override.Tolerations
	}
}

func replaceAffinities(override *base.WorkloadOverride, ps *corev1.PodTemplateSpec) {
	if override.Affinity != nil {
		ps.Spec.Affinity = override.Affinity
	}
}

func replaceResources(override *base.WorkloadOverride, ps *corev1.PodTemplateSpec) {
	if len(override.Resources) > 0 {
		containers := ps.Spec.Containers
		for i := range containers {
			if override := findResourceOverride(override.Resources, containers[i].Name); override != nil {
				merge(&override.Limits, &containers[i].Resources.Limits)
				merge(&override.Requests, &containers[i].Resources.Requests)
			}
		}
	}
}

func replaceEnv(override *base.WorkloadOverride, ps *corev1.PodTemplateSpec) {
	if len(override.Env) > 0 {
		containers := ps.Spec.Containers
		for i := range containers {
			if override := findEnvOverride(override.Env, containers[i].Name); override != nil {
				mergeEnv(&override.EnvVars, &containers[i].Env)
			}
		}
	}
}

func replaceProbes(override *base.WorkloadOverride, ps *corev1.PodTemplateSpec) {
	if len(override.ReadinessProbes) > 0 {
		containers := ps.Spec.Containers
		for i := range containers {
			override := findProbeOverride(override.ReadinessProbes, containers[i].Name)
			if override != nil {
				overrideProbe := &corev1.Probe{
					InitialDelaySeconds:           override.InitialDelaySeconds,
					TimeoutSeconds:                override.TimeoutSeconds,
					PeriodSeconds:                 override.PeriodSeconds,
					SuccessThreshold:              override.SuccessThreshold,
					FailureThreshold:              override.FailureThreshold,
					TerminationGracePeriodSeconds: override.TerminationGracePeriodSeconds,
				}
				if *overrideProbe == (corev1.Probe{}) {
					//  Disable probe when users explicitly set the empty overrideProbe.
					containers[i].ReadinessProbe = nil
					continue
				}
				if containers[i].ReadinessProbe == nil {
					containers[i].ReadinessProbe = overrideProbe
					continue
				}
				mergeProbe(overrideProbe, containers[i].ReadinessProbe)
			}
		}
	}

	if len(override.LivenessProbes) > 0 {
		containers := ps.Spec.Containers
		for i := range containers {
			if override := findProbeOverride(override.LivenessProbes, containers[i].Name); override != nil {
				overrideProbe := &corev1.Probe{
					InitialDelaySeconds:           override.InitialDelaySeconds,
					TimeoutSeconds:                override.TimeoutSeconds,
					PeriodSeconds:                 override.PeriodSeconds,
					SuccessThreshold:              override.SuccessThreshold,
					FailureThreshold:              override.FailureThreshold,
					TerminationGracePeriodSeconds: override.TerminationGracePeriodSeconds,
				}
				if *overrideProbe == (corev1.Probe{}) {
					//  Disable probe when users explicitly set the empty overrideProbe.
					containers[i].LivenessProbe = nil
					continue
				}
				if containers[i].LivenessProbe == nil {
					containers[i].LivenessProbe = overrideProbe
					continue
				}
				mergeProbe(overrideProbe, containers[i].LivenessProbe)
			}
		}
	}
}

func replaceHostNetwork(override *base.WorkloadOverride, ps *corev1.PodTemplateSpec) {
	if override.HostNetwork != nil {
		ps.Spec.HostNetwork = *override.HostNetwork

		if *override.HostNetwork {
			ps.Spec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
		}
	}
}
