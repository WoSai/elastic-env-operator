package v1alpha1

import (
	"github.com/wosai/elastic-env-operator/api/v1alpha1"
	"k8s.io/client-go/rest"
)

type (
	ElasticEnvOperatorV1alpha1Interface interface {
		RESTClient() rest.Interface
		SQBApplicationGetter
		SQBDeploymentGetter
		SQBPlaneGetter
	}

	// ElasticEnvOperatorV1alpha1Client is used to interact with features provided by the qa.shouqianba.com group.
	ElasticEnvOperatorV1alpha1Client struct {
		restClient rest.Interface
	}
)

const (
	applicationResource = "sqbapplications"
	deploymentResource  = "sqbdeployments"
	planeResource       = "sqbplanes"
)

func (ec *ElasticEnvOperatorV1alpha1Client) SQBApplication(ns string) SQBApplicationInterface {
	return newSQBApplication(ec, ns)
}

func (ec *ElasticEnvOperatorV1alpha1Client) SQBDeployment(ns string) SQBDeploymentInterface {
	return newSQBDeployment(ec, ns)
}

func (ec *ElasticEnvOperatorV1alpha1Client) SQBPlane(ns string) SQBPlaneInterface {
	return newSQBPlane(ec, ns)
}

func (ec *ElasticEnvOperatorV1alpha1Client) RESTClient() rest.Interface {
	return ec.restClient
}

// NewForConfig creates a new ElasticEnvOperatorV1alpha1Client for given config
func NewForConfig(c *rest.Config) (*ElasticEnvOperatorV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ElasticEnvOperatorV1alpha1Client{restClient: client}, nil
}

// NewForConfigOrDie creates a new ElasticEnvV1alpha1Client for the given config and panics
// if  there is an error in config
func NewForConfigOrDie(c *rest.Config) *ElasticEnvOperatorV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ElasticEnvV1alphaClient for the given RESTClient
func New(c rest.Interface) *ElasticEnvOperatorV1alpha1Client {
	return &ElasticEnvOperatorV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	config.GroupVersion = &v1alpha1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	return nil
}
