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