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
	// SQBApplicationGetter has a method to return a SQBApplicationInterface
	SQBApplicationGetter interface {
		SQBApplication(namespace string) SQBApplicationInterface
	}

	// SQBApplicationInterface has methods to work with SQBApplication resources.
	SQBApplicationInterface interface {
		Create(context.Context, *v1alpha1.SQBApplication, v1.CreateOptions) (*v1alpha1.SQBApplication, error)
		Update(context.Context, *v1alpha1.SQBApplication, v1.UpdateOptions) (*v1alpha1.SQBApplication, error)
		//UpdateStatus(ctx context.Context, application *v1alpha1.SQBApplication, opts v1.UpdateOptions) (*v1alpha1.SQBApplication, error)
		Delete(context.Context, string, v1.DeleteOptions) error
		DeleteCollection(context.Context, v1.DeleteOptions, v1.ListOptions) error
		Get(ctx context.Context, name string, opt v1.GetOptions) (*v1alpha1.SQBApplication, error)
		List(context.Context, v1.ListOptions) (*v1alpha1.SQBApplicationList, error)
		Watch(context.Context, v1.ListOptions) (watch.Interface, error)
		Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opt v1.PatchOptions, subresources ...string) (*v1alpha1.SQBApplication, error)
	}

	// sqbApplication implements SQBApplicationInterface
	sqbApplication struct {
		client    rest.Interface
		namespace string
	}
)

func newSQBApplication(c *ElasticEnvOperatorV1alpha1Client, namespace string) *sqbApplication {
	return &sqbApplication{
		client:    c.RESTClient(),
		namespace: namespace,
	}
}

// Get takes name of the SQBApplication, and returns the corresponding SQBApplication, and on error if there is any.
func (c *sqbApplication) Get(ctx context.Context, name string, opt v1.GetOptions) (*v1alpha1.SQBApplication, error) {
	result := &v1alpha1.SQBApplication{}
	err := c.client.Get().
		Namespace(c.namespace).
		Resource(applicationResource).
		Name(name).
		VersionedParams(&opt, ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

// List takes label and field selectors, and returns the list of SQBApplication that match those selector
func (c *sqbApplication) List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.SQBApplicationList, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result := &v1alpha1.SQBApplicationList{}
	err := c.client.Get().Namespace(c.namespace).Resource(applicationResource).VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).Do(ctx).Into(result)
	return result, err
}

// Watch returns a watch.Interface that watches requested SQBApplication
func (c *sqbApplication) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().Namespace(c.namespace).Resource(applicationResource).VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).Watch(ctx)
}

// Create takes the representation of a SQBApplication and creates it, returns the server's representation
// of the SQBApplication, and an error, if there is any.
func (c *sqbApplication) Create(ctx context.Context, app *v1alpha1.SQBApplication, opts v1.CreateOptions) (result *v1alpha1.SQBApplication, err error) {
	result = &v1alpha1.SQBApplication{}
	err = c.client.Post().Namespace(c.namespace).Resource(applicationResource).VersionedParams(&opts, ParameterCodec).
		Body(app).Do(ctx).Into(result)
	return
}

// Update takes the representation of a SQBApplication and updates it. Returns the server;s representation of
// the SQBApplication, and an error, if there is any.
func (c *sqbApplication) Update(ctx context.Context, app *v1alpha1.SQBApplication, opts v1.UpdateOptions) (result *v1alpha1.SQBApplication, err error) {
	result = &v1alpha1.SQBApplication{}
	err = c.client.Put().Namespace(c.namespace).Resource(applicationResource).Namespace(app.Name).
		VersionedParams(&opts, ParameterCodec).Body(app).Do(ctx).Into(result)
	return
}

// Delete takes name of the SQBApplication and deletes it. returns an error if there is any
func (c *sqbApplication) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().Namespace(c.namespace).Resource(applicationResource).Name(name).Body(&opts).Do(ctx).Error()
}

// DeleteCollection deletes a collection of objects.
func (c *sqbApplication) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().Namespace(c.namespace).Resource(applicationResource).Timeout(timeout).
		VersionedParams(&listOpts, ParameterCodec).Body(&opts).Do(ctx).Error()
}

func (c *sqbApplication) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (*v1alpha1.SQBApplication, error) {
	result := &v1alpha1.SQBApplication{}
	err := c.client.Patch(pt).Namespace(c.namespace).SubResource(subresources...).
		VersionedParams(&opts, ParameterCodec).Body(data).Do(ctx).Into(result)
	return result, err
}
