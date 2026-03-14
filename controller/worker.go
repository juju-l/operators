package controller

import (
	"fmt"
	"sync"

	"tst-operator/apis/example.com/v1beta1"

	"k8s.io/client-go/tools/cache"
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

	// 加对象锁
	lock, _ := c.objectLocks.LoadOrStore(key, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	defer func() {
		lock.(*sync.Mutex).Unlock()
		c.objectLocks.Delete(key)
	}()

	// 执行 reconcile
	if err := c.reconcile(key); err != nil {
		// 重试
		c.queue.AddRateLimited(key)
	} else {
		c.queue.Forget(obj)
	}
	return true
}

// reconcile 核心逻辑
func (c *Controller) reconcile(key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	// 1. 获取资源
	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Printf("tst %s/%s deleted\n", ns, name)
		return nil
	}

	tst := obj.(*v1beta1.Tst)
	// ==============================
	// 关键：资源版本校验（检测是否变更）
	// ==============================
	latestRV := tst.ResourceVersion
	fmt.Printf("reconcile %s/%s, RV: %s\n", ns, name, latestRV)

	// 2. 业务逻辑（示例：判断副本数）
	newStatus := tst.Status.DeepCopy()
	newStatus.ReadyReplicas = tst.Spec.Replicas
	newStatus.State = "Active"
	newStatus.Message = "operator managed"

	// 3. Patch 更新 Status（仅合并，不覆盖）
	// 传入 RV，用于外部二次校验
	if err := c.PatchTstStatus(ns, name, latestRV, newStatus); err != nil {
		return err
	}

	return nil
}