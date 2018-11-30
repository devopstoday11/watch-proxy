package v1alpha3

import (
	v1alpha3 "github.com/heptio/quartermaster/custom/apis/virtualservice/v1alpha3"
	"github.com/heptio/quartermaster/custom/client/clientset/versioned/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type VirtualserviceV1alpha3Interface interface {
	RESTClient() rest.Interface
	VirtualServicesGetter
}

// VirtualserviceV1alpha3Client is used to interact with features provided by the virtualservice group.
type VirtualserviceV1alpha3Client struct {
	restClient rest.Interface
}

func (c *VirtualserviceV1alpha3Client) VirtualServices(namespace string) VirtualServiceInterface {
	return newVirtualServices(c, namespace)
}

// NewForConfig creates a new VirtualserviceV1alpha3Client for the given config.
func NewForConfig(c *rest.Config) (*VirtualserviceV1alpha3Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &VirtualserviceV1alpha3Client{client}, nil
}

// NewForConfigOrDie creates a new VirtualserviceV1alpha3Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *VirtualserviceV1alpha3Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new VirtualserviceV1alpha3Client for the given RESTClient.
func New(c rest.Interface) *VirtualserviceV1alpha3Client {
	return &VirtualserviceV1alpha3Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha3.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *VirtualserviceV1alpha3Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}