package controller

import (
	"time"

	"your/hlm-operator/hlm"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
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