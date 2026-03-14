package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Explanation: 完善 controller，处理 unstructured 的 spec/hash 计算、变更检测、status patch 的冲突重试和上下文超时

type Controller struct {
	DynClient dynamic.Interface
	GVR       schema.GroupVersionResource
	Queue     workqueue.RateLimitingInterface
	Informer  cache.SharedIndexInformer
}

func NewController(dyn dynamic.Interface, gvr schema.GroupVersionResource, informer cache.SharedIndexInformer) *Controller {
	q := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "tst-controller")
	c := &Controller{
		DynClient: dyn,
		GVR:       gvr,
		Queue:     q,
		Informer:  informer,
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				klog.Warning("add: object is not Unstructured")
				return
			}
			key, err := cache.MetaNamespaceKeyFunc(u)
			if err != nil {
				klog.Warningf("failed to get key: %v", err)
				return
			}
			c.Queue.Add(key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newU, ok := newObj.(*unstructured.Unstructured)
			if !ok {
				klog.Warning("update: object is not Unstructured")
				return
			}
			oldU, ok2 := oldObj.(*unstructured.Unstructured)
			if !ok2 {
				klog.Warning("update: old object is not Unstructured")
				return
			}
			if oldU.GetResourceVersion() == newU.GetResourceVersion() {
				return
			}
			key, err := cache.MetaNamespaceKeyFunc(newU)
			if err != nil {
				klog.Warningf("failed to get key: %v", err)
				return
			}
			c.Queue.Add(key)
		},
		DeleteFunc: func(obj interface{}) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
					if t, ok2 := tombstone.Obj.(*unstructured.Unstructured); ok2 {
						u = t
					} else {
						klog.Warning("delete tombstone contained non-Unstructured object")
						return
					}
				} else {
					klog.Warning("delete: object is not Unstructured")
					return
				}
			}
			key, err := cache.MetaNamespaceKeyFunc(u)
			if err != nil {
				klog.Warningf("failed to get key: %v", err)
				return
			}
			c.Queue.Add(key)
		},
	})

	return c
}

func (c *Controller) Run(ctx context.Context, workers int) {
	defer c.Queue.ShutDown()
	klog.Info("starting controller")
	go c.Informer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), c.Informer.HasSynced) {
		klog.Error("timed out waiting for caches to sync")
		return
	}

	for i := 0; i < workers; i++ {
		go func() {
			for {
				item, shutdown := c.Queue.Get()
				if shutdown {
					return
				}
				key := item.(string)
				if err := c.processKey(ctx, key); err != nil {
					klog.Errorf("error processing %s: %v", key, err)
					c.Queue.AddRateLimited(key)
				} else {
					c.Queue.Forget(item)
				}
				c.Queue.Done(item)
			}
		}()
	}

	<-ctx.Done()
	klog.Info("stopping controller")
}

func (c *Controller) processKey(ctx context.Context, key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	res := c.DynClient.Resource(c.GVR).Namespace(ns)

	// use a per-invocation timeout
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	obj, err := res.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("object %s/%s not found", ns, name)
			return nil
		}
		// transient error -> requeue
		klog.Warningf("failed to get object %s/%s: %v", ns, name, err)
		return err
	}

	// handle deletion
	if obj.GetDeletionTimestamp() != nil {
		klog.Infof("object %s/%s is being deleted", ns, name)
		// cleanup if finalizer exists
		return nil
	}

	// compute spec hash
	var s string
	spec, found, err := unstructured.NestedFieldCopy(obj.Object, "spec")
	if err != nil {
		klog.Warningf("error getting spec: %v", err)
	}
	if !found {
		s = ""
	} else {
		b, _ := json.Marshal(spec)
		h := sha256.Sum256(b)
		s = hex.EncodeToString(h[:])
	}

	// read status observedGeneration & lastHandledSpecHash
	obsGen, _, _ := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	lastHash, _, _ := unstructured.NestedString(obj.Object, "status", "lastHandledSpecHash")

	if obj.GetGeneration() == obsGen && lastHash == s {
		// nothing to do
		klog.Infof("no change for %s/%s", ns, name)
		return nil
	}

	// perform business logic here (create/update dependent resources)
	klog.Infof("reconciling %s/%s", ns, name)

	// update status via patch with retry on conflict
	patch := map[string]interface{}{
		"status": map[string]interface{}{
			"observedGeneration": obj.GetGeneration(),
			"lastHandledSpecHash": s,
			"conditions": []map[string]interface{}{{
				"type": "Ready",
				"status": "True",
				"lastTransitionTime": time.Now().Format(time.RFC3339),
			}} ,
		},
	}
	patchBytes, _ := json.Marshal(patch)

	var lastErr error
	for i := 0; i < 5; i++ {
		_, err = res.Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
		if err == nil {
			klog.Infof("status patched for %s/%s", ns, name)
			return nil
		}
		lastErr = err
		klog.Warningf("failed patch status attempt %d: %v", i+1, err)
		// on conflict, re-get and rebuild minimal patch
		if i < 4 {
			time.Sleep(time.Duration(i*i) * 100 * time.Millisecond)
			newObj, gerr := res.Get(ctx, name, metav1.GetOptions{})
			if gerr != nil {
				klog.Warningf("failed get while retrying patch: %v", gerr)
				continue
			}
			// rebuild patch to be minimal
			patch = map[string]interface{}{
				"status": map[string]interface{}{
					"observedGeneration": newObj.GetGeneration(),
					"lastHandledSpecHash": s,
				},
			}
			patchBytes, _ = json.Marshal(patch)
		}
	}

	return fmt.Errorf("failed to patch status after retries: %w", lastErr)
}
