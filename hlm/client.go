package hlm

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

func NewForConfig(c *rest.Config) (*HlmClient, error) {
	config := *c
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(
		runtime.NewScheme(),
	).WithoutConversion()

	rc, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &HlmClient{restClient: rc}, nil
}

type HlmClient struct {
	restClient rest.Interface
}

func (c *HlmClient) RESTClient() rest.Interface {
	return c.restClient
}

func (c *HlmClient) Hlms(namespace string) HlmInterface {
	return &hlmNamespaceClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

type HlmInterface interface {
	Get(opt1 any, s string, opts metav1.GetOptions) (*Hlm, error)
	Update(ctx any, hlm *Hlm, opts metav1.UpdateOptions) (*Hlm, error)
	UpdateStatus(ctx any, hlm *Hlm, opts metav1.UpdateOptions) (*Hlm, error)
}

type hlmNamespaceClient struct {
	restClient rest.Interface
	ns         string
}

func (c *hlmNamespaceClient) Get(
	ctx context.Context,
	name string,
	opts metav1.GetOptions,
) (*Hlm, error) {
	result := &Hlm{}
	err := c.restClient.Get().
		Namespace(c.ns).
		Resource("hlms").
		Name(name).
		VersionedParams(&opts, metav1.ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *hlmNamespaceClient) Update(
	ctx context.Context,
	obj *Hlm,
	opts metav1.UpdateOptions,
) (*Hlm, error) {
	result := &Hlm{}
	err := c.restClient.Put().
		Namespace(c.ns).
		Resource("hlms").
		Name(obj.Name).
		Body(obj).
		VersionedParams(&opts, metav1.ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *hlmNamespaceClient) UpdateStatus(
	ctx context.Context,
	obj *Hlm,
	opts metav1.UpdateOptions,
) (*Hlm, error) {
	result := &Hlm{}
	err := c.restClient.Put().
		Namespace(c.ns).
		Resource("hlms").
		Name(obj.Name).
		SubResource("status").
		Body(obj).
		VersionedParams(&opts, metav1.ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}