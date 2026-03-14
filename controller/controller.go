package controller

import (
	"context"
	"sync"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"tst-operator/client"
)

type Controller struct {
	ctx       context.Context
	cancel    context.CancelFunc
	clientset *client.TstClient
	informer  cache.SharedIndexInformer
	queue     workqueue.RateLimitingInterface
	// 对象级锁：防止同一资源并发 reconcile
	objectLocks sync.Map
	// 并发 worker 数
	workerCount int
}

func NewController(
	ctx context.Context,
	clientset *client.TstClient,
	informer cache.SharedIndexInformer,
	workerCount int,
) *Controller {
	ctx, cancel := context.WithCancel(ctx)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// 事件注册：增/删/改 全部入队
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, _ := cache.MetaNamespaceKeyFunc(obj)
			queue.Add(key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, _ := cache.MetaNamespaceKeyFunc(newObj)
			queue.Add(key)
		},
		DeleteFunc: func(obj interface{}) {
			key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			queue.Add(key)
		},
	})

	return &Controller{
		ctx:         ctx,
		cancel:      cancel,
		clientset:   clientset,
		informer:    informer,
		queue:       queue,
		workerCount: workerCount,
	}
}

// Start 启动 controller & worker
func (c *Controller) Start() {
	go c.informer.Run(c.ctx.Done())

	// 等待缓存同步
	if !cache.WaitForCacheSync(c.ctx.Done(), c.informer.HasSynced) {
		panic("informer sync failed")
	}

	// 启动多 worker
	for i := 0; i < c.workerCount; i++ {
		go c.runWorker()
	}

	<-c.ctx.Done()
}

func (c *Controller) Stop() {
	c.cancel()
	c.queue.ShutDown()
}