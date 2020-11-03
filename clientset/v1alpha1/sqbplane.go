package v1alpha1

import (
	"context"
	"time"

	"github.com/wosai/elastic-env-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type (
	// SQBPlaneGetter has a method to return a SQBPlaneInterface
	SQBPlaneGetter interface {
		SQBPlane(namespace string) SQBPlaneInterface
	}

	// SQBPlaneInterface has methods to work with SQBPlane resources.
	SQBPlaneInterface interface {
		Create(context.Context, *v1alpha1.SQBPlane, v1.CreateOptions) (*v1alpha1.SQBPlane, error)
		Update(context.Context, *v1alpha1.SQBPlane, v1.UpdateOptions) (*v1alpha1.SQBPlane, error)
		Delete(context.Context, string, v1.DeleteOptions) error
		DeleteCollection(context.Context, v1.DeleteOptions, v1.ListOptions) error
		Get(context.Context, string, v1.GetOptions) (*v1alpha1.SQBPlane, error)
		List(context.Context, v1.ListOptions) (*v1alpha1.SQBPlaneList, error)
		Watch(context.Context, v1.ListOptions) (watch.Interface, error)
		Patch(context.Context, string, types.PatchType, []byte, v1.PatchOptions, ...string) (*v1alpha1.SQBPlane, error)
	}

	// sqbPlane implements SQBPlaneInterface
	sqbPlane struct {
		client    rest.Interface
		namespace string
	}
)

func newSQBPlane(c *ElasticEnvOperatorV1alpha1Client, namespace string) *sqbPlane {
	return &sqbPlane{
		client:    c.RESTClient(),
		namespace: namespace,
	}
}

// Get takes name of the SQBPlane, and returns the corresponding SQBPlane, and on error if there is any.
func (c *sqbPlane) Get(ctx context.Context, name string, opt v1.GetOptions) (*v1alpha1.SQBPlane, error) {
	result := &v1alpha1.SQBPlane{}
	err := c.client.Get().
		Namespace(c.namespace).
		Resource(planeResource).
		Name(name).
		VersionedParams(&opt, ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

// List takes label and field selectors, and returns the list of SQBPlane that match those selector
func (c *sqbPlane) List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.SQBPlaneList, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result := &v1alpha1.SQBPlaneList{}
	err := c.client.Get().Namespace(c.namespace).Resource(planeResource).VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).Do(ctx).Into(result)
	return result, err
}

// Watch returns a watch.Interface that watches requested SQBPlane
func (c *sqbPlane) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().Namespace(c.namespace).Resource(planeResource).VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).Watch(ctx)
}

// Create takes the representation of a SQBPlane and creates it, returns the server's representation
// of the SQBPlane, and an error, if there is any.
func (c *sqbPlane) Create(ctx context.Context, plane *v1alpha1.SQBPlane, opts v1.CreateOptions) (result *v1alpha1.SQBPlane, err error) {
	result = &v1alpha1.SQBPlane{}
	err = c.client.Post().Namespace(c.namespace).Resource(planeResource).VersionedParams(&opts, ParameterCodec).
		Body(plane).Do(ctx).Into(result)
	return
}

// Update takes the representation of a SQBPlane and updates it. Returns the server;s representation of
// the SQBPlane, and an error, if there is any.
func (c *sqbPlane) Update(ctx context.Context, plane *v1alpha1.SQBPlane, opts v1.UpdateOptions) (result *v1alpha1.SQBPlane, err error) {
	result = &v1alpha1.SQBPlane{}
	err = c.client.Put().Namespace(c.namespace).Resource(planeResource).Name(plane.Name).
		VersionedParams(&opts, ParameterCodec).Body(plane).Do(ctx).Into(result)
	return
}

// Delete takes name of the SQBPlane and deletes it. returns an error if there is any
func (c *sqbPlane) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().Namespace(c.namespace).Resource(planeResource).Name(name).Body(&opts).Do(ctx).Error()
}

// DeleteCollection deletes a collection of objects.
func (c *sqbPlane) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().Namespace(c.namespace).Resource(planeResource).Timeout(timeout).
		VersionedParams(&listOpts, ParameterCodec).Body(&opts).Do(ctx).Error()
}

// Patch applies the patch and returns the patched SQBPlane
func (c *sqbPlane) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (*v1alpha1.SQBPlane, error) {
	result := &v1alpha1.SQBPlane{}
	err := c.client.Patch(pt).Namespace(c.namespace).Resource(planeResource).SubResource(subresources...).Name(name).
		VersionedParams(&opts, ParameterCodec).Body(data).Do(ctx).Into(result)
	return result, err
}
