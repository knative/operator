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

package base

import (
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

const (
	// DependenciesInstalled is a Condition indicating that potential dependencies have
	// been installed correctly.
	DependenciesInstalled apis.ConditionType = "DependenciesInstalled"
	// InstallSucceeded is a Condition indicating that the installation of the component
	// itself has been successful.
	InstallSucceeded apis.ConditionType = "InstallSucceeded"
	// DeploymentsAvailable is a Condition indicating whether or not the Deployments of
	// the respective component have come up successfully.
	DeploymentsAvailable apis.ConditionType = "DeploymentsAvailable"
	// VersionMigrationEligible is a Condition indicating whether or not the current version of
	// Knative component is eligible to upgrade or downgrade to the specified version.
	VersionMigrationEligible apis.ConditionType = "VersionMigrationEligible"
)

// KComponent is a common interface for accessing meta, spec and status of all known types.
type KComponent interface {
	metav1.Object
	schema.ObjectKind

	// GetSpec returns the common spec for all known types.
	GetSpec() KComponentSpec
	// GetStatus returns the common status of all known types.
	GetStatus() KComponentStatus
}

// KComponentSpec is a common interface for accessing the common spec of all known types.
type KComponentSpec interface {
	// GetConfig returns means to override entries in upstream configmaps.
	GetConfig() ConfigMapData
	// GetRegistry returns means to override deployment images.
	GetRegistry() *Registry
	// GetResources returns a list of container resource overrides.
	GetResources() []ResourceRequirementsOverride
	// GetVersion gets the version to be installed
	GetVersion() string
	// GetManifests gets the list of manifests, which should ultimately be installed
	GetManifests() []Manifest
	// GetAdditionalManifests gets the list of additional manifests, which should be installed
	GetAdditionalManifests() []Manifest

	// GetHighAvailability returns means to set the number of desired replicas
	GetHighAvailability() *HighAvailability

	// GetWorkloadOverrides gets the component configurations to override.
	GetWorkloadOverrides() []WorkloadOverride

	// GetServiceOverride gets the service configurations to override.
	GetServiceOverride() []ServiceOverride

	// GetPodDisruptionBudgetOverride gets the PodDisruptionBudget configurations to override.
	GetPodDisruptionBudgetOverride() []PodDisruptionBudgetOverride
}

// KComponentStatus is a common interface for status mutations of all known types.
type KComponentStatus interface {
	// MarkInstallSucceeded marks the InstallationSucceeded status as true.
	MarkInstallSucceeded()
	// MarkInstallFailed marks the InstallationSucceeded status as false with the given
	// message.
	MarkInstallFailed(msg string)

	// MarkDeploymentsAvailable marks the DeploymentsAvailable status as true.
	MarkDeploymentsAvailable()
	// MarkDeploymentsNotReady marks the DeploymentsAvailable status as false and calls out
	// it's waiting for deployments.
	MarkDeploymentsNotReady([]string)

	// MarkVersionMigrationEligible marks the VersionMigrationEligible status as true.
	MarkVersionMigrationEligible()
	// MarkVersionMigrationNotEligible marks the VersionMigrationEligible status as false with
	// the given message.
	MarkVersionMigrationNotEligible(msg string)

	// MarkDependenciesInstalled marks the DependenciesInstalled status as true.
	MarkDependenciesInstalled()
	// MarkDependencyInstalling marks the DependenciesInstalled status as false with the
	// given message.
	MarkDependencyInstalling(msg string)
	// MarkDependencyMissing marks the DependenciesInstalled status as false with the
	// given message.
	MarkDependencyMissing(msg string)

	// GetVersion gets the currently installed version of the component.
	GetVersion() string
	// SetVersion sets the currently installed version of the component.
	SetVersion(version string)

	// GetManifests gets the url links of the manifests
	GetManifests() []string
	// SetManifests sets the url links of the manifests
	SetManifests(manifests []string)

	// IsReady return true if all conditions are satisfied
	IsReady() bool
}

// CommonSpec unifies common fields and functions on the Spec.
type CommonSpec struct {
	// A means to override the corresponding entries in the upstream configmaps
	// +optional
	Config ConfigMapData `json:"config,omitempty"`

	// A means to override the corresponding deployment images in the upstream.
	// If no registry is provided, the knative release images will be used.
	// +optional
	Registry Registry `json:"registry,omitempty"`

	// DEPRECATED.
	// DeprecatedResources overrides containers' resource requirements.
	// +optional
	DeprecatedResources []ResourceRequirementsOverride `json:"resources,omitempty"`

	// DEPRECATED. Use components
	// DeploymentOverride overrides Deployment configurations such as resources and replicas.
	// +optional
	DeploymentOverride []WorkloadOverride `json:"deployments,omitempty"`

	// Workloads overrides workloads configurations such as resources and replicas.
	// +optional
	Workloads []WorkloadOverride `json:"workloads,omitempty"`

	// ServiceOverride overrides Service configurations such as labels and annotations.
	// +optional
	ServiceOverride []ServiceOverride `json:"services,omitempty"`

	// WorkloadOverride containers' resource requirements
	// +optional
	Version string `json:"version,omitempty"`

	// A means to specify the manifests to install
	// +optional
	Manifests []Manifest `json:"manifests,omitempty"`

	// A means to specify the additional manifests to install
	// +optional
	AdditionalManifests []Manifest `json:"additionalManifests,omitempty"`

	// HighAvailability allows specification of HA control plane.
	// +optional
	HighAvailability *HighAvailability `json:"high-availability,omitempty"`

	// PodDisruptionBudgetOverride overrides PodDisruptionBudget configurations via minAvailable.
	// +optional
	PodDisruptionBudgetOverride []PodDisruptionBudgetOverride `json:"podDisruptionBudgets,omitempty"`
}

// GetConfig implements KComponentSpec.
func (c *CommonSpec) GetConfig() ConfigMapData {
	return c.Config
}

// GetRegistry implements KComponentSpec.
func (c *CommonSpec) GetRegistry() *Registry {
	return &c.Registry
}

// GetResources implements KComponentSpec.
func (c *CommonSpec) GetResources() []ResourceRequirementsOverride {
	return c.DeprecatedResources
}

// GetVersion implements KComponentSpec.
func (c *CommonSpec) GetVersion() string {
	return c.Version
}

// GetManifests implements KComponentSpec.
func (c *CommonSpec) GetManifests() []Manifest {
	return c.Manifests
}

// GetAdditionalManifests implements KComponentSpec.
func (c *CommonSpec) GetAdditionalManifests() []Manifest {
	return c.AdditionalManifests
}

// GetHighAvailability implements KComponentSpec.
func (c *CommonSpec) GetHighAvailability() *HighAvailability {
	return c.HighAvailability
}

// GetDeploymentOverride implements KComponentSpec.
func (c *CommonSpec) GetWorkloadOverrides() []WorkloadOverride {
	return append(c.DeploymentOverride, c.Workloads...)
}

// GetServiceOverride implements KComponentSpec.
func (c *CommonSpec) GetServiceOverride() []ServiceOverride {
	return c.ServiceOverride
}

// GetPodDisruptionBudgetOverride implements KComponentSpec.
func (c *CommonSpec) GetPodDisruptionBudgetOverride() []PodDisruptionBudgetOverride {
	return c.PodDisruptionBudgetOverride
}

// ConfigMapData is a nested map of maps representing all upstream ConfigMaps. The first
// level key is the key to the ConfigMap itself (i.e. "logging") while the second level
// is the data to be filled into the respective ConfigMap.
type ConfigMapData map[string]map[string]string

// Registry defines image overrides of knative images.
// This affects both apps/v1.Deployment and caching.internal.knative.dev/v1beta1.Image.
// The default value is used as a default format to override for all knative deployments.
// The override values are specific to each knative deployment.
type Registry struct {
	// The default image reference template to use for all knative images.
	// It takes the form of example-registry.io/custom/path/${NAME}:custom-tag
	// ${NAME} will be replaced by the deployment container name, or caching.internal.knative.dev/v1alpha1/Image name.
	// +optional
	Default string `json:"default,omitempty"`

	// A map of a container name or image name to the full image location of the individual knative image.
	// +optional
	Override map[string]string `json:"override,omitempty"`

	// A list of secrets to be used when pulling the knative images. The secret must be created in the
	// same namespace as the knative-serving deployments, and not the namespace of this resource.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// WorkloadOverride defines the configurations of deployments to override.
type WorkloadOverride struct {
	// Name is the name of the deployment to override.
	Name string `json:"name"`

	// Labels overrides labels for the deployment and its template.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations overrides labels for the deployment and its template.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Replicas is the number of replicas that HA parts of the control plane
	// will be scaled to.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// NodeSelector overrides nodeSelector for the deployment.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations overrides tolerations for the deployment.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Affinities overrides affinity for the deployment.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Resources overrides resources for the containers.
	// +optional
	Resources []ResourceRequirementsOverride `json:"resources,omitempty"`

	// Env overrides env vars for the containers.
	// +optional
	Env []EnvRequirementsOverride `json:"env,omitempty"`

	// ReadinessProbes overrides readiness probes for the containers.
	// +optional
	ReadinessProbes []ProbesRequirementsOverride `json:"readinessProbes,omitempty"`

	// LivenessProbes overrides liveness probes for the containers.
	// +optional
	LivenessProbes []ProbesRequirementsOverride `json:"livenessProbes,omitempty"`
}

// ServiceOverride defines the configurations of the service to override.
type ServiceOverride struct {
	// Name is the name of the service to override.
	Name string `json:"name"`

	// Labels overrides labels for the service and its template.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations overrides labels for the service and its template.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Selector overrides the selector for the service
	// +optional
	Selector map[string]string `json:"selector,omitempty"`
}

type PodDisruptionBudgetOverride struct {
	// Name is the name of the podDisruptionBudget to override.
	Name string `json:"name"`
	// The desired PodDisruptionBudgetSpec
	policyv1.PodDisruptionBudgetSpec
}

// ResourceRequirementsOverride enables the user to override any container's
// resource requests/limits specified in the embedded manifest
type ResourceRequirementsOverride struct {
	// The container name
	Container string `json:"container"`
	// The desired ResourceRequirements
	corev1.ResourceRequirements
}

// EnvRequirementsOverride enables the user to override any container's env vars.
type EnvRequirementsOverride struct {
	// The container name
	Container string `json:"container"`
	// The desired EnvVarRequirements
	EnvVars []corev1.EnvVar `json:"envVars,omitempty"`
}

// ProbesRequirementsOverride enables the user to override any container's env vars.
type ProbesRequirementsOverride struct {
	// The container name
	Container string `json:"container"`
	// Number of seconds after the container has started before liveness probes are initiated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty" protobuf:"varint,2,opt,name=initialDelaySeconds"`
	// Number of seconds after which the probe times out.
	// Defaults to 1 second. Minimum value is 1.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,3,opt,name=timeoutSeconds"`
	// How often (in seconds) to perform the probe.
	// Default to 10 seconds. Minimum value is 1.
	// +optional
	PeriodSeconds int32 `json:"periodSeconds,omitempty" protobuf:"varint,4,opt,name=periodSeconds"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.
	// +optional
	SuccessThreshold int32 `json:"successThreshold,omitempty" protobuf:"varint,5,opt,name=successThreshold"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Defaults to 3. Minimum value is 1.
	// +optional
	FailureThreshold int32 `json:"failureThreshold,omitempty" protobuf:"varint,6,opt,name=failureThreshold"`
	// Optional duration in seconds the pod needs to terminate gracefully upon probe failure.
	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// Set this value longer than the expected cleanup time for your process.
	// If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this
	// value overrides the value provided by the pod spec.
	// Value must be non-negative integer. The value zero indicates stop immediately via
	// the kill signal (no opportunity to shut down).
	// This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate.
	// Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty" protobuf:"varint,7,opt,name=terminationGracePeriodSeconds"`
}

// Manifest enables the user to specify the links to the manifests' URLs
type Manifest struct {
	// The link of the manifest URL
	Url string `json:"URL"`
}

// HighAvailability specifies options for deploying Knative Serving control
// plane in a highly available manner. Note that HighAvailability is still in
// progress and does not currently provide a completely HA control plane.
type HighAvailability struct {
	// Replicas is the number of replicas that HA parts of the control plane
	// will be scaled to.
	Replicas *int32 `json:"replicas"`
}

// CustomCerts refers to either a ConfigMap or Secret containing valid
// CA certificates
type CustomCerts struct {
	// One of ConfigMap or Secret
	Type string `json:"type"`

	// The name of the ConfigMap or Secret
	Name string `json:"name"`
}
