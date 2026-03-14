package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Explanation: 实现确定性 spec 序列化与哈希、增强调试日志、status patch 冲突处理（重获取并重新计算 specHash），以及 finalizer 删除清理逻辑。

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

// stableMarshal produces deterministic JSON for hashing using ordered key/value pairs
func stableMarshal(v interface{}) ([]byte, error) {
	c := canonicalize(v)
	return json.Marshal(c)
}

// canonicalize converts maps into ordered slices of {k,v} objects to ensure deterministic
// JSON output independent of Go map iteration order.
func canonicalize(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]interface{}, 0, len(keys))
		for _, k := range keys {
			out = append(out, map[string]interface{}{"k": k, "v": canonicalize(t[k])})
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i := range t {
			out[i] = canonicalize(t[i])
		}
		return out
	default:
		return t
	}
}

const finalizerName = "tst.example.com/finalizer"

func (c *Controller) processKey(ctx context.Context, key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	res := c.DynClient.Resource(c.GVR).Namespace(ns)

	// per-invocation timeout
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	obj, err := res.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("object %s/%s not found", ns, name)
			return nil
		}
		klog.Warningf("failed to get object %s/%s: %v", ns, name, err)
		return err
	}

	// Ensure finalizer exists for objects we manage
	finalizers := obj.GetFinalizers()
	if !containsString(finalizers, finalizerName) && obj.GetDeletionTimestamp() == nil {
		klog.Infof("adding finalizer to %s/%s", ns, name)
		newFinalizers := append(finalizers, finalizerName)
		metaPatch := map[string]interface{}{"metadata": map[string]interface{}{"finalizers": newFinalizers}}
		patchBytes, _ := json.Marshal(metaPatch)
		_, err := res.Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			klog.Warningf("failed to add finalizer for %s/%s: %v", ns, name, err)
			return err
		}
		// refetch object for up-to-date state
		obj, err = res.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			klog.Warningf("failed to get object after adding finalizer %s/%s: %v", ns, name, err)
			return err
		}
	}

	// Deletion handling: if deletionTimestamp set, run finalizer cleanup if present
	if obj.GetDeletionTimestamp() != nil {
		finalizers := obj.GetFinalizers()
		if containsString(finalizers, finalizerName) {
			klog.Infof("object %s/%s being deleted and has finalizer, running cleanup", ns, name)
			// perform cleanup (idempotent)
			if err := c.cleanupAssociatedResources(ctx, obj); err != nil {
				klog.Warningf("cleanup failed for %s/%s: %v", ns, name, err)
				return err
			}
			// remove finalizer
			newFinalizers := removeString(finalizers, finalizerName)
			metaPatch := map[string]interface{}{"metadata": map[string]interface{}{"finalizers": newFinalizers}}
			patchBytes, _ := json.Marshal(metaPatch)
			_, err := res.Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
			if err != nil {
				klog.Warningf("failed to remove finalizer for %s/%s: %v", ns, name, err)
				return err
			}
			klog.Infof("finalizer removed for %s/%s", ns, name)
		}
		// nothing else to do
		return nil
	}

	// compute deterministic spec hash
	var s string
	spec, found, err := unstructured.NestedFieldCopy(obj.Object, "spec")
	if err != nil {
		klog.Warningf("error getting spec for %s/%s: %v", ns, name, err)
	}
	if !found {
		s = ""
	} else {
		stable, merr := stableMarshal(spec)
		if merr != nil {
			klog.Warningf("stable marshal failed for %s/%s: %v", ns, name, merr)
			b, _ := json.Marshal(spec)
			h := sha256.Sum256(b)
			s = hex.EncodeToString(h[:])
		} else {
			h := sha256.Sum256(stable)
			s = hex.EncodeToString(h[:])
		}
	}

	// read status observedGeneration & lastHandledSpecHash
	obsGen, _, _ := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	lastHash, _, _ := unstructured.NestedString(obj.Object, "status", "lastHandledSpecHash")

	// detailed debug log
	klog.Infof("debug reconcile %s/%s: generation=%d observedGeneration=%d lastHandledSpecHash=%s computedSpecHash=%s resourceVersion=%s", ns, name, obj.GetGeneration(), obsGen, lastHash, s, obj.GetResourceVersion())

	if obj.GetGeneration() == obsGen && lastHash == s {
		klog.Infof("no change for %s/%s", ns, name)
		return nil
	}

	// business logic placeholder
	klog.Infof("reconciling %s/%s", ns, name)

	// prepare minimal status patch
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
		klog.Warningf("failed patch status attempt %d for %s/%s: %v", i+1, ns, name, err)
		// on conflict, re-get and decide
		if apierrors.IsConflict(err) && i < 4 {
			// short backoff
			time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
			newObj, gerr := res.Get(ctx, name, metav1.GetOptions{})
			if gerr != nil {
				klog.Warningf("failed get while retrying patch for %s/%s: %v", ns, name, gerr)
				continue
			}
			// recompute spec hash from latest object
			newSpec, nfound, _ := unstructured.NestedFieldCopy(newObj.Object, "spec")
			var newHash string
			if !nfound {
				newHash = ""
			} else {
				stable, merr := stableMarshal(newSpec)
				if merr != nil {
					b, _ := json.Marshal(newSpec)
					h := sha256.Sum256(b)
					newHash = hex.EncodeToString(h[:])
				} else {
					h := sha256.Sum256(stable)
					newHash = hex.EncodeToString(h[:])
				}
			}
			if newHash != s {
				// spec changed meanwhile; abort status write to avoid overwrite and requeue
				klog.Infof("spec changed for %s/%s during patch retries: oldHash=%s newHash=%s; requeueing", ns, name, s, newHash)
				return fmt.Errorf("spec changed during patch retries")
			}
			// if spec unchanged, rebuild minimal patch using latest generation
			patch = map[string]interface{}{
				"status": map[string]interface{}{
					"observedGeneration": newObj.GetGeneration(),
					"lastHandledSpecHash": s,
				},
			}
			patchBytes, _ = json.Marshal(patch)
		}
		// other errors will retry
		if i < 4 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	return fmt.Errorf("failed to patch status after retries: %w", lastErr)
}

func (c *Controller) cleanupAssociatedResources(ctx context.Context, obj *unstructured.Unstructured) error {
	// TODO: 实现实际的清理逻辑（删除下游资源、释放外部资源等）。此处为占位并返回 nil 表示清理成功。
	// 在真实实现中应保证幂等性并记录事件/日志。
	klog.Infof("cleanupAssociatedResources placeholder for %s/%s", obj.GetNamespace(), obj.GetName())
	return nil
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	out := make([]string, 0, len(slice))
	for _, v := range slice {
		if v == s {
			continue
		}
		out = append(out, v)
	}
	return out
}
