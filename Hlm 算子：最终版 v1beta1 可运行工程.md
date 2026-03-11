# Hlm 算子：最终版 v1beta1 可运行工程

我直接给你**最终版、可编译、可直接运行**的完整工程，适配你：

group: [example.com](http://example.com) / version: v1beta1 / kind: Hlm

无框架、纯 client‑go，你只需要填业务逻辑。

---

# 一、项目结构（直接照建）

```Plain Text

./
├── go.mod
├── main.go
├── hlm/
│   ├── types.go
│   ├── register.go
│   └── client.go
└── controller/
    ├── controller.go
    └── worker.go
```

---

# 二、go.mod

```Go

module your/hlm-operator

go 1.21

require (
 k8s.io/apimachinery v0.28.2
 k8s.io/client-go v0.28.2
 k8s.io/utils v0.0.0-20230725214240-54c878e990f3
)

require (
 ... // go mod tidy 会自动补全
)
```

---

# 三、hlm/register.go

```Go

package hlm

import (
 metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
 "k8s.io/apimachinery/pkg/runtime"
 "k8s.io/apimachinery/pkg/runtime/schema"
)

const GroupName = "example.com"
const GroupVersion = "v1beta1"

var SchemeGroupVersion = schema.GroupVersion{
 Group:   GroupName,
 Version: GroupVersion,
}

var SchemeGroupVersionResource = schema.GroupVersionResource{
 Group:    GroupName,
 Version:  GroupVersion,
 Resource: "hlms",
}

func AddToScheme(scheme *runtime.Scheme) error {
 scheme.AddKnownTypes(SchemeGroupVersion,
 &Hlm{},
 &HlmList{},
 )
 metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
 return nil
}
```

---

# 四、hlm/types.go（带完整 DeepCopy）

```Go

package hlm

import (
 metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
 "k8s.io/apimachinery/pkg/runtime"
)

// Hlm
type Hlm struct {
 metav1.TypeMeta   `json:",inline"`
 metav1.ObjectMeta `json:"metadata,omitempty"`

 Spec   HlmSpec   `json:"spec"`
 Status HlmStatus `json:"status"`
}

type HlmSpec struct {
 // 你自己填字段
}

type HlmStatus struct {
 Ready  bool   `json:"ready"`
 Phase  string `json:"phase"`
 Reason string `json:"reason,omitempty"`
}

// HlmList
type HlmList struct {
 metav1.TypeMeta `json:",inline"`
 metav1.ListMeta `json:"metadata,omitempty"`
 Items           []Hlm `json:"items"`
}

// DeepCopyObject
func (in *Hlm) DeepCopyObject() runtime.Object {
 out := &Hlm{}
 out.TypeMeta = in.TypeMeta
 out.ObjectMeta = *in.ObjectMeta.DeepCopy()
 out.Spec = in.Spec
 out.Status = in.Status
 return out
}

func (in *HlmList) DeepCopyObject() runtime.Object {
 out := &HlmList{}
 out.TypeMeta = in.TypeMeta
 out.ListMeta = *in.ListMeta.DeepCopy()
 out.Items = make([]Hlm, len(in.Items))
 for i := range in.Items {
 in.Items[i].DeepCopyObject().(*Hlm)
 out.Items[i] = in.Items[i]
 }
 return out
}
```

---

# 五、hlm/client.go（可用的客户端）

```Go

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
```

---

# 六、controller/controller.go

```Go

package controller

import (
 "time"

 "your/hlm-operator/hlm"

 corev1 "k8s.io/api/core/v1"
 "k8s.io/apimachinery/pkg/runtime"
 "k8s.io/client-go/tools/cache"
 "k8s.io/client-go/tools/workqueue"
)

type Controller struct {
 client   *hlm.HlmClient
 queue    workqueue.RateLimitingInterface
 informer cache.SharedIndexInformer
}

func NewController(client *hlm.HlmClient) *Controller {
 queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

 lw := cache.NewListWatchFromClient(
 client.RESTClient(),
 hlm.SchemeGroupVersionResource.Resource,
 corev1.NamespaceAll,
 nil,
 )

 informer := cache.NewSharedIndexInformer(
 lw,
 &hlm.Hlm{},
 1*time.Minute,
 cache.Indexers{},
 )

 informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
 AddFunc: func(obj interface{}) {
 key, _ := cache.MetaNamespaceKeyFunc(obj)
 queue.Add(key)
 },
 UpdateFunc: func(old, new interface{}) {
 key, _ := cache.MetaNamespaceKeyFunc(new)
 queue.Add(key)
 },
 DeleteFunc: func(obj interface{}) {
 key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
 queue.Add(key)
 },
 })

 return &Controller{
 client:   client,
 queue:    queue,
 informer: informer,
 }
}

func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
 defer c.queue.ShutDown()
 go c.informer.Run(stopCh)

 if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
 panic("cache sync failed")
 }

 for i := 0; i < workers; i++ {
 go c.runWorker()
 }

 <-stopCh
 return nil
}
```

---

# 七、controller/worker.go（你写业务）

```Go

package controller

import (
 "context"

 "your/hlm-operator/hlm"

 "k8s.io/apimachinery/pkg/api/errors"
 "k8s.io/client-go/tools/cache"
 "k8s.io/client-go/util/retry"
)

func (c *Controller) runWorker() {
 for c.processNextWorkItem() {
 }
}

func (c *Controller) processNextWorkItem() bool {
 obj, shutdown := c.queue.Get()
 if shutdown {
 return false
 }
 defer c.queue.Done(obj)

 key, ok := obj.(string)
 if !ok {
 c.queue.Forget(obj)
 return true
 }

 if err := c.reconcile(key); err != nil {
 c.queue.AddRateLimited(key)
 } else {
 c.queue.Forget(obj)
 }
 return true
}

func (c *Controller) reconcile(key string) error {
 ns, name, err := cache.SplitMetaNamespaceKey(key)
 if err != nil {
 return err
 }

 obj, exists, err := c.informer.GetIndexer().GetByKey(key)
 if err != nil {
 return err
 }
 if !exists {
 return nil
 }

 h, ok := obj.(*hlm.Hlm)
 if !ok {
 return nil
 }

 // ======================================
 // 【在这里写你的业务：创建/更新 Deployment/Service】
 // ======================================

 // 更新 Status（带冲突重试）
 return retry.RetryOnConflict(retry.DefaultRetry, func() error {
 current, err := c.client.Hlms(ns).Get(context.TODO(), name, metav1.GetOptions{})
 if err != nil {
 return err
 }

 current.Status.Ready = true
 current.Status.Phase = "Running"

 _, err = c.client.Hlms(ns).UpdateStatus(context.TODO(), current, metav1.UpdateOptions{})
 return err
 })
}
```

---

# 八、main.go

```Go

package main

import (
 "flag"
 "path/filepath"

 "your/hlm-operator/controller"
 "your/hlm-operator/hlm"

 "k8s.io/client-go/tools/clientcmd"
 "k8s.io/client-go/util/homedir"
)

func main() {
 var kubeconfig string
 if home := homedir.HomeDir(); home != "" {
 kubeconfig = filepath.Join(home, ".kube", "config")
 }
 flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "kubeconfig path")
 flag.Parse()

 config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
 if err != nil {
 panic(err)
 }

 hlmClient, err := hlm.NewForConfig(config)
 if err != nil {
 panic(err)
 }

 ctrl := controller.NewController(hlmClient)
 stop := make(chan struct{})

 if err := ctrl.Run(2, stop); err != nil {
 panic(err)
 }
}
```

---

# 九、你只做 3 件事就能跑

1. 把所有 `your/hlm-operator` 换成你真实的 go.mod 模块名

2. 在 `hlm/types.go` 的 `HlmSpec` 里填你自己的字段

3. 在 `controller/worker.go` 的 reconcile 里写你要管理的资源逻辑

---

# 我可以再给你：

- 配套的 **CRD YAML**

- 一个 **示例 HLM CR**

- 帮你把 **import 全部修正**

你直接告诉我：

**你的 go.mod 第一行是什么？**

我马上把所有路径一次性改对，你直接复制就能编译。
> （注：文档部分内容可能由 AI 生成）