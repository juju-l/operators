package helms

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

// Controller Operator 控制器
type Controller struct {
	dynamic    dynamic.Interface
	queue      workqueue.RateLimitingInterface
	informer   cache.SharedIndexInformer
	restConfig *rest.Config
	helmOp     *HelmOperator
}

// NewController 创建控制器
func NewController(kubeconfig, chartPath string) (*Controller, error) {
	var cfg *rest.Config
	var err error
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    "tpl.vipex.cc",
		Version:  "v1alpha1",
		Resource: "hlm",
	}

	inf := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				return dyn.Resource(gvr).List(context.TODO(), opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				return dyn.Resource(gvr).Watch(context.TODO(), opts)
			},
		},
		&Hlm{},
		time.Minute,
		cache.Indexers{},
	)

	q := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, _ := cache.MetaNamespaceKeyFunc(obj)
			q.Add(key)
		},
		UpdateFunc: func(_, newObj interface{}) {
			key, _ := cache.MetaNamespaceKeyFunc(newObj)
			q.Add(key)
		},
		DeleteFunc: func(obj interface{}) {
			key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			q.Add(key)
		},
	})

	return &Controller{
		dynamic:    dyn,
		queue:      q,
		informer:   inf,
		restConfig: cfg,
		helmOp:     NewHelmOperator(chartPath),
	}, nil
}

// Run 启动控制器
func (c *Controller) Run(ctx context.Context) {
	defer c.queue.ShutDown()
	go c.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		log.Fatal("sync failed")
	}
	log.Println("controller started")

	for i := 0; i < 2; i++ {
		go c.worker(ctx)
	}
	<-ctx.Done()
	log.Println("stopped")
}

func (c *Controller) worker(ctx context.Context) {
	for c.process(ctx) {
	}
}

func (c *Controller) process(ctx context.Context) bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(obj)

	key, ok := obj.(string)
	if !ok {
		return true
	}

	if err := c.reconcile(ctx, key); err != nil {
		c.queue.AddRateLimited(key)
		log.Printf("reconcile error: %v", err)
	} else {
		c.queue.Forget(obj)
	}
	return true
}

// reconcile 核心逻辑（适配 Helm v4，传入 ctx）
func (c *Controller) reconcile(ctx context.Context, key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		return c.helmOp.Uninstall(ctx, name, ns, c.restConfig)
	}

	hlm := obj.(*Hlm)
	// 传入 ctx，适配 Helm v4 的 Run 方法
	_, err = c.helmOp.InstallOrUpgrade(ctx, &hlm.Spec, c.restConfig)
	return err
}

// main 程序入口
func main() {
	var kc string
	var chart string
	flag.StringVar(&kc, "kubeconfig", "", "kubeconfig path (optional)")
	flag.StringVar(&chart, "chart", "./tpl_vipex_cc-0.1.0.tgz", "path to helm chart tgz")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	ctrl, err := NewController(kc, chart)
	if err != nil {
		log.Fatal(err)
	}
	ctrl.Run(ctx)
}
