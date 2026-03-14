package client

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	"tst-operator/apis/example.com/v1beta1"
)

// 保留原有接口定义
type TstClientInterface interface {
	Tsts(namespace string) TstInterface
}

type TstInterface interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1beta1.Tst, error)
	Create(ctx context.Context, Tst *v1beta1.Tst, opts metav1.CreateOptions) (*v1beta1.Tst, error)
	Update(ctx context.Context, Tst *v1beta1.Tst, opts metav1.UpdateOptions) (*v1beta1.Tst, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1beta1.Tst, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1beta1.TstList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

// ========== 核心：仅保留核心字段，无任何自定义配置 ==========
type TstClient struct {
	restClient rest.Interface
	codec      runtime.ParameterCodec // 本地保存编码配置，不依赖 rest.Config
}

func NewForConfig(c *rest.Config) (*TstClient, error) {
	// 1. 初始化自定义 Scheme（所有版本通用）
	scheme := runtime.NewScheme()
	if err := v1beta1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add scheme: %v", err)
	}

	// 2. 配置序列化器（所有版本通用）
	codecs := serializer.NewCodecFactory(scheme)
	negotiatedSerializer := codecs.WithoutConversion()

	// 3. 仅配置 rest.Config 的核心必填字段（所有版本都有）
	config := *c
	config.GroupVersion = &v1beta1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = negotiatedSerializer
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	// 4. 创建 restClient（仅依赖核心字段）
	rc, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to create rest client: %v", err)
	}

	// 5. 初始化参数编码（本地保存，不依赖 rest.Config）
	codec := runtime.NewParameterCodec(scheme)

	return &TstClient{
		restClient: rc,
		codec:      codec,
	}, nil
}

// 传递本地 codec 到 Tst 结构体
func (c *TstClient) Tsts(namespace string) TstInterface {
	return &Tst{
		restClient: c.restClient,
		codec:      c.codec,
		ns:         namespace,
	}
}

// Tst 结构体：仅保存核心依赖
type Tst struct {
	restClient rest.Interface
	codec      runtime.ParameterCodec // 本地 codec
	ns         string
}

// ========== 所有方法：直接使用本地 codec，不依赖 restClient 配置 ==========
// List 方法（终极修复：无任何 rest.Config 字段依赖）
func (c *Tst) List(ctx context.Context, opts metav1.ListOptions) (*v1beta1.TstList, error) {
	result := &v1beta1.TstList{}
	req := c.restClient.Get().Resource("Tsts")
	if c.ns != metav1.NamespaceAll {
		req = req.Namespace(c.ns)
	}

	// 关键：直接使用本地 codec 编码，完全不依赖 rest.Config 的任何字段
	err := req.
		VersionedParams(&opts, c.codec).
		Do(ctx).
		Into(result)
	return result, err
}

// Watch 方法（同 List，无依赖）
func (c *Tst) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	req := c.restClient.Get().Resource("tsts")
	if c.ns != metav1.NamespaceAll {
		req = req.Namespace(c.ns)
	}

	return req.
		VersionedParams(&opts, c.codec).
		Watch(ctx)
}

// Get 方法（无依赖）
func (c *Tst) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1beta1.Tst, error) {
	result := &v1beta1.Tst{}
	err := c.restClient.Get().
		Namespace(c.ns).
		Resource("tsts").
		Name(name).
		VersionedParams(&opts, c.codec).
		Do(ctx).
		Into(result)
	return result, err
}

// Create 方法（无依赖）
func (c *Tst) Create(ctx context.Context, Tst *v1beta1.Tst, opts metav1.CreateOptions) (*v1beta1.Tst, error) {
	result := &v1beta1.Tst{}
	err := c.restClient.Post().
		Namespace(c.ns).
		Resource("tsts").
		VersionedParams(&opts, c.codec).
		Body(Tst).
		Do(ctx).
		Into(result)
	return result, err
}

// Update 方法（无依赖）
func (c *Tst) Update(ctx context.Context, Tst *v1beta1.Tst, opts metav1.UpdateOptions) (*v1beta1.Tst, error) {
	result := &v1beta1.Tst{}
	err := c.restClient.Put().
		Namespace(c.ns).
		Resource("tsts").
		Name(Tst.Name).
		VersionedParams(&opts, c.codec).
		Body(Tst).
		Do(ctx).
		Into(result)
	return result, err
}

// Delete 方法（无依赖）
func (c *Tst) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.restClient.Delete().
		Namespace(c.ns).
		Resource("tsts").
		Name(name).
		VersionedParams(&opts, c.codec).
		Do(ctx).
		Error()
}

// Patch 方法（无依赖）
func (c *Tst) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1beta1.Tst, error) {
	result := &v1beta1.Tst{}
	err := c.restClient.Patch(pt).
		Namespace(c.ns).
		Resource("tsts").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, c.codec).
		Body(data).
		Do(ctx).
		Into(result)
	return result, err
}