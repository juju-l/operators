package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

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
			meta := obj.(metav1.Object)
			key := meta.GetNamespace() + "/" + meta.GetName()
			c.Queue.Add(key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldMeta := oldObj.(metav1.Object)
			newMeta := newObj.(metav1.Object)
			if oldMeta.GetResourceVersion() == newMeta.GetResourceVersion() {
				return
			}
			key := newMeta.GetNamespace() + "/" + newMeta.GetName()
			c.Queue.Add(key)
		},
		DeleteFunc: func(obj interface{}) {
			meta := obj.(metav1.Object)
			key := meta.GetNamespace() + "/" + meta.GetName()
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
	obj, err := res.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		klog.Infof("get error %v", err)
		return nil
	}

	// handle deletion
	if obj.GetDeletionTimestamp() != nil {
		klog.Infof("object %s/%s is being deleted", ns, name)
		// cleanup if finalizer exists
		return nil
	}

	// compute spec hash
	spec, found, err := unstructured.NestedFieldCopy(obj.Object, "spec")
	if err != nil || !found {
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

	// update status via patch
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
	_, err = res.Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		klog.Errorf("failed patch status: %v", err)
		return err
	}

	return nil
}
