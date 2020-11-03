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
	// SQBDeploymentGetter has a method to return a SQBDeploymentInterface
	SQBDeploymentGetter interface {
		SQBDeployment(namespace string) SQBDeploymentInterface
	}

	// SQBDeploymentInterface has methods to work with SQBDeployment resources.
	SQBDeploymentInterface interface {
		Create(context.Context, *v1alpha1.SQBDeployment, v1.CreateOptions) (*v1alpha1.SQBDeployment, error)
		Update(context.Context, *v1alpha1.SQBDeployment, v1.UpdateOptions) (*v1alpha1.SQBDeployment, error)
		Delete(context.Context, string, v1.DeleteOptions) error
		DeleteCollection(context.Context, v1.DeleteOptions, v1.ListOptions) error
		Get(context.Context, string, v1.GetOptions) (*v1alpha1.SQBDeployment, error)
		List(context.Context, v1.ListOptions) (*v1alpha1.SQBDeploymentList, error)
		Watch(context.Context, v1.ListOptions) (watch.Interface, error)
		Patch(context.Context, string, types.PatchType, []byte, v1.PatchOptions, ...string) (*v1alpha1.SQBDeployment, error)
	}

	// sqbDeployment implements SQBDeploymentInterface
	sqbDeployment struct {
		client    rest.Interface
		namespace string
	}
)

func newSQBDeployment(c *ElasticEnvOperatorV1alpha1Client, namespace string) *sqbDeployment {
	return &sqbDeployment{
		client:    c.RESTClient(),
		namespace: namespace,
	}
}

// Get takes name of the SQBDeployment, and returns the corresponding SQBDeployment, and on error if there is any.
func (c *sqbDeployment) Get(ctx context.Context, name string, opt v1.GetOptions) (*v1alpha1.SQBDeployment, error) {
	result := &v1alpha1.SQBDeployment{}
	err := c.client.Get().
		Namespace(c.namespace).
		Resource(deploymentResource).
		Name(name).
		VersionedParams(&opt, ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

// List takes label and field selectors, and returns the list of SQBDeployment that match those selector
func (c *sqbDeployment) List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.SQBDeploymentList, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result := &v1alpha1.SQBDeploymentList{}
	err := c.client.Get().Namespace(c.namespace).Resource(deploymentResource).VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).Do(ctx).Into(result)
	return result, err
}

// Watch returns a watch.Interface that watches requested SQBDeployment
func (c *sqbDeployment) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().Namespace(c.namespace).Resource(deploymentResource).VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).Watch(ctx)
}

// Create takes the representation of a SQBDeployment and creates it, returns the server's representation
// of the SQBDeployment, and an error, if there is any.
func (c *sqbDeployment) Create(ctx context.Context, deployment *v1alpha1.SQBDeployment, opts v1.CreateOptions) (result *v1alpha1.SQBDeployment, err error) {
	result = &v1alpha1.SQBDeployment{}
	err = c.client.Post().Namespace(c.namespace).Resource(deploymentResource).VersionedParams(&opts, ParameterCodec).
		Body(deployment).Do(ctx).Into(result)
	return
}

// Update takes the representation of a SQBDeployment and updates it. Returns the server;s representation of
// the SQBDeployment, and an error, if there is any.
func (c *sqbDeployment) Update(ctx context.Context, deployment *v1alpha1.SQBDeployment, opts v1.UpdateOptions) (result *v1alpha1.SQBDeployment, err error) {
	result = &v1alpha1.SQBDeployment{}
	err = c.client.Put().Namespace(c.namespace).Resource(deploymentResource).
		VersionedParams(&opts, ParameterCodec).Body(deployment).Do(ctx).Into(result)
	return
}

// Delete takes name of the SQBDeployment and deletes it. returns an error if there is any
func (c *sqbDeployment) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().Namespace(c.namespace).Resource(deploymentResource).Name(name).Body(&opts).Do(ctx).Error()
}

// DeleteCollection deletes a collection of objects.
func (c *sqbDeployment) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().Namespace(c.namespace).Resource(deploymentResource).Timeout(timeout).
		VersionedParams(&listOpts, ParameterCodec).Body(&opts).Do(ctx).Error()
}

// Patch applies the patch and returns the patched SQBDeployment
func (c *sqbDeployment) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (*v1alpha1.SQBDeployment, error) {
	result := &v1alpha1.SQBDeployment{}
	err := c.client.Patch(pt).Namespace(c.namespace).SubResource(subresources...).Name(name).Resource(deploymentResource).
		VersionedParams(&opts, ParameterCodec).Body(data).Do(ctx).Into(result)
	return result, err
}
