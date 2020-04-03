package client

import (
	mf "github.com/manifestival/manifestival"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func NewManifest(pathname string, config *rest.Config, opts ...mf.Option) (mf.Manifest, error) {
	client, err := NewClient(config)
	if err != nil {
		return mf.Manifest{}, err
	}
	return mf.NewManifest(pathname, append(opts, mf.UseClient(client))...)
}

func NewClient(config *rest.Config) (mf.Client, error) {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	mapper, err := restmapper.NewDynamicRESTMapper(config)
	if err != nil {
		return nil, err
	}
	return &clientGoClient{client: client, mapper: mapper}, nil
}

type clientGoClient struct {
	client dynamic.Interface
	mapper meta.RESTMapper
}

// verify implementation
var _ mf.Client = (*clientGoClient)(nil)

func (c *clientGoClient) Create(obj *unstructured.Unstructured, options ...mf.ApplyOption) error {
	resource, err := c.resourceInterface(obj)
	if err != nil {
		return err
	}
	opts := mf.ApplyWith(options)
	_, err = resource.Create(obj, *opts.ForCreate)
	return err
}

func (c *clientGoClient) Update(obj *unstructured.Unstructured, options ...mf.ApplyOption) error {
	resource, err := c.resourceInterface(obj)
	if err != nil {
		return err
	}
	opts := mf.ApplyWith(options)
	_, err = resource.Update(obj, *opts.ForUpdate)
	return err
}

func (c *clientGoClient) Delete(obj *unstructured.Unstructured, options ...mf.DeleteOption) error {
	resource, err := c.resourceInterface(obj)
	if err != nil {
		return err
	}
	opts := mf.DeleteWith(options)
	err = resource.Delete(obj.GetName(), opts.ForDelete)
	if apierrors.IsNotFound(err) && opts.IgnoreNotFound {
		return nil
	}
	return err
}

func (c *clientGoClient) Get(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	resource, err := c.resourceInterface(obj)
	if err != nil {
		return nil, err
	}
	return resource.Get(obj.GetName(), metav1.GetOptions{})
}

func (c *clientGoClient) resourceInterface(obj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	gvk := obj.GroupVersionKind()
	mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		return c.client.Resource(mapping.Resource), nil
	}
	return c.client.Resource(mapping.Resource).Namespace(obj.GetNamespace()), nil
}
