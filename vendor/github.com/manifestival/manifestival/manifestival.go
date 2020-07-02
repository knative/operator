package manifestival

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testing"
	"github.com/manifestival/manifestival/pkg/client"
	"github.com/manifestival/manifestival/pkg/dry"
	"github.com/manifestival/manifestival/pkg/filter"
	"github.com/manifestival/manifestival/pkg/overlay"
	"github.com/manifestival/manifestival/pkg/patch"
	"github.com/manifestival/manifestival/pkg/sources"
	"github.com/manifestival/manifestival/pkg/transform"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Manifestival defines the operations allowed on a set of Kubernetes
// resources (typically, a set of YAML files, aka a manifest)
type Manifestival interface {
	// Either updates or creates all resources in the manifest
	Apply(opts ...client.ApplyOption) error
	// Deletes all resources in the manifest
	Delete(opts ...client.DeleteOption) error
	// Transforms the resources within a Manifest
	Transform(fns ...transform.Transformer) (Manifest, error)
	// Filters resources in a Manifest; Predicates are AND'd
	Filter(fns ...filter.Predicate) Manifest
	// Append the resources from other Manifests to create a new one
	Append(mfs ...Manifest) Manifest
	// Show how applying the manifest would change the cluster
	DryRun() ([]dry.MergePatch, error)
}

// Manifest tracks a set of concrete resources which should be managed as a
// group using a Kubernetes client
type Manifest struct {
	resources []unstructured.Unstructured
	Client    client.Client
	log       logr.Logger
}

// Option enables configuration of your Manifest
type Option func(*Manifest)

// UseLogger will cause manifestival to log its actions
func UseLogger(log logr.Logger) Option {
	return func(m *Manifest) {
		m.log = log
	}
}

// UseClient enables interaction with the k8s API server
func UseClient(client client.Client) Option {
	return func(m *Manifest) {
		m.Client = client
	}
}

var _ Manifestival = &Manifest{}

// NewManifest creates a Manifest from a comma-separated set of YAML
// files, directories, or URLs. It's equivalent to
// `ManifestFrom(Path(pathname))`
func NewManifest(pathname string, opts ...Option) (Manifest, error) {
	return ManifestFrom(sources.Path(pathname), opts...)
}

// ManifestFrom creates a Manifest from any Source implementation
func ManifestFrom(src sources.Source, opts ...Option) (m Manifest, err error) {
	m = Manifest{log: testing.NullLogger{}}
	for _, opt := range opts {
		opt(&m)
	}
	m.log.Info("Parsing manifest")
	m.resources, err = src.Parse()
	return
}

// Append creates a new Manifest by appending the resources from other
// Manifests onto this one. No equality checking is done, so for any
// resources sharing the same GVK+name, the last one will "win".
func (m Manifest) Append(mfs ...Manifest) Manifest {
	result := m
	result.resources = m.Resources() // deep copies
	for _, mf := range mfs {
		result.resources = append(result.resources, mf.Resources()...)
	}
	return result
}

// Resources returns a deep copy of the Manifest resources
func (m Manifest) Resources() []unstructured.Unstructured {
	result := make([]unstructured.Unstructured, len(m.resources))
	for i, v := range m.resources {
		result[i] = *v.DeepCopy()
	}
	return result
}

// Apply updates or creates all resources in the manifest.
func (m Manifest) Apply(opts ...client.ApplyOption) error {
	for _, spec := range m.resources {
		if err := m.apply(&spec, opts...); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes all resources in the Manifest
func (m Manifest) Delete(opts ...client.DeleteOption) error {
	a := make([]unstructured.Unstructured, len(m.resources))
	copy(a, m.resources) // shallow copy is fine
	// we want to delete in reverse order
	for left, right := 0, len(a)-1; left < right; left, right = left+1, right-1 {
		a[left], a[right] = a[right], a[left]
	}
	for _, spec := range a {
		if okToDelete(&spec) {
			if err := m.delete(&spec, opts...); err != nil {
				return err
			}
		}
	}
	return nil
}

// Filter returns a new, immutable Manifest containing only the
// resources for which *all* Predicates return true. Any changes
// callers make to the resources passed to their Predicate[s] will
// only be reflected in the returned Manifest.
func (m Manifest) Filter(preds ...filter.Predicate) Manifest {
	result := m
	result.resources = filter.Filter(m.resources, preds...)
	return result
}

// Transform returns a new, immutable Manifest resulting from the
// application of an ordered set of Transformer functions to the
// `Resources` in this Manifest.
func (m Manifest) Transform(fns ...transform.Transformer) (result Manifest, err error) {
	result = m
	result.resources, err = transform.Transform(m.resources, fns...)
	return
}

// DryRun returns a list of merge patches, either strategic or
// RFC-7386 for unregistered types, that show the effects of applying
// the manifest
func (m Manifest) DryRun() ([]dry.MergePatch, error) {
	return dry.DryRun(m.resources, m.Client)
}

// apply updates or creates a particular resource
func (m Manifest) apply(spec *unstructured.Unstructured, opts ...client.ApplyOption) error {
	current, err := m.get(spec)
	if err != nil {
		return err
	}
	if current == nil {
		m.logResource("Creating", spec)
		current = spec.DeepCopy()
		annotate(current, v1.LastAppliedConfigAnnotation, lastApplied(current))
		annotate(current, "manifestival", resourceCreated)
		return m.Client.Create(current, opts...)
	} else {
		diff, err := patch.New(current, spec)
		if err != nil {
			return err
		}
		if diff == nil {
			return nil
		}
		m.log.Info("Merging", "diff", diff)
		if err := diff.Merge(current); err != nil {
			return err
		}
		return m.update(current, spec, opts...)
	}
}

// update a single resource
func (m Manifest) update(live, spec *unstructured.Unstructured, opts ...client.ApplyOption) error {
	m.logResource("Updating", live)
	annotate(live, v1.LastAppliedConfigAnnotation, lastApplied(spec))
	err := m.Client.Update(live, opts...)
	if errors.IsInvalid(err) && client.ApplyWith(opts).Overwrite {
		m.log.Error(err, "Failed to update merged resource, trying overwrite")
		overlay.Copy(spec.Object, live.Object)
		return m.Client.Update(live, opts...)
	}
	return err
}

// delete removes the specified object
func (m Manifest) delete(spec *unstructured.Unstructured, opts ...client.DeleteOption) error {
	current, err := m.get(spec)
	if current == nil && err == nil {
		return nil
	}
	m.logResource("Deleting", spec)
	return m.Client.Delete(spec, opts...)
}

// get collects a full resource body (or `nil`) from a partial
// resource supplied in `spec`
func (m Manifest) get(spec *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	result, err := m.Client.Get(spec)
	if err != nil {
		result = nil
		if errors.IsNotFound(err) {
			err = nil
		}
	}
	return result, err
}

// logResource logs a consistent formatted message
func (m Manifest) logResource(msg string, spec *unstructured.Unstructured) {
	name := fmt.Sprintf("%s/%s", spec.GetNamespace(), spec.GetName())
	m.log.Info(msg, "name", name, "type", spec.GroupVersionKind())
}

// annotate sets an annotation in the resource
func annotate(spec *unstructured.Unstructured, key string, value string) {
	annotations := spec.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	spec.SetAnnotations(annotations)
}

// lastApplied returns a JSON string denoting the resource's state
func lastApplied(obj *unstructured.Unstructured) string {
	ann := obj.GetAnnotations()
	if len(ann) > 0 {
		delete(ann, v1.LastAppliedConfigAnnotation)
		obj.SetAnnotations(ann)
	}
	bytes, _ := obj.MarshalJSON()
	return string(bytes)
}

// okToDelete checks for an annotation indicating that the resources
// was originally created by this library
func okToDelete(spec *unstructured.Unstructured) bool {
	switch spec.GetKind() {
	case "Namespace":
		return spec.GetAnnotations()["manifestival"] == resourceCreated
	}
	return true
}

const (
	resourceCreated = "new"
)
