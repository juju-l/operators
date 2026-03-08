package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
)

var operatorGVR = schema.GroupVersionResource{Group: "tpl.vipex.cc", Version: "v1alpha1", Resource: "hlms"}
var finalizerName = "hlm.finalizers.tpl.vipex.cc"

// Controller watches Hlm resources and reconciles them using Helm
type Controller struct {
	dynClient dynamic.Interface
	kubeClient kubernetes.Interface
	namespace string
	recorder record.EventRecorder
	helm *HelmClient

	// Patch retry configuration
	MaxPatchAttempts int           // max attempts for status patch retries
	PatchBaseDelay   time.Duration // base delay for exponential backoff
}

// NewController creates controller using in-cluster config if possible
func NewController(kubeconfig string, namespace string, helm *HelmClient) (*Controller, error) {
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

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Event recorder: needs a scheme; using a basic scheme
	scheme := runtime.NewScheme()
	// note: in a fuller implementation you would add corev1 and your CRD types to the scheme
	broadcaster := record.NewBroadcaster()
	rec := broadcaster.NewRecorder(scheme, corev1.EventSource{Component: "hlm-operator"})

	c := &Controller{
		dynClient: dyn,
		kubeClient: kubeClient,
		helm: helm,
		recorder: rec,
		namespace: namespace,
		// defaults; may be overridden by env
		MaxPatchAttempts: 5,
		PatchBaseDelay: 200 * time.Millisecond,
	}

	// Allow overrides via environment variables
	if v := os.Getenv("PATCH_MAX_ATTEMPTS"); v != "" {
		if vi, err := strconv.Atoi(v); err == nil && vi > 0 {
			c.MaxPatchAttempts = vi
		}
	}
	if v := os.Getenv("PATCH_BASE_DELAY_MS"); v != "" {
		if mi, err := strconv.Atoi(v); err == nil && mi > 0 {
			c.PatchBaseDelay = time.Duration(mi) * time.Millisecond
		}
	}

	return c, nil
}

// Run starts the controller loop
func (c *Controller) Run(ctx context.Context, stopCh <-chan struct{}) error {
	resClient := c.dynClient.Resource(operatorGVR).Namespace(c.namespace)

	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return resClient.List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return resClient.Watch(ctx, options)
		},
	}

	_, controller := cache.NewInformer(lw, &unstructured.Unstructured{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.handle(obj, "ADD")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.handle(newObj, "UPDATE")
		},
		DeleteFunc: func(obj interface{}) {
			// informer Delete may be final state; still pass to handle
			c.handle(obj, "DELETE")
		},
	})

	go controller.Run(stopCh)

	<-stopCh
	return nil
}

// patchStatusForObject uses server-side apply with exponential backoff retries; falls back to MergePatch with retries on failure
func (c *Controller) patchStatusForObject(ctx context.Context, u *unstructured.Unstructured, status HlmStatus) error {
	bs, err := json.Marshal(status)
	if err != nil {
		return err
	}
	var sm map[string]interface{}
	if err := json.Unmarshal(bs, &sm); err != nil {
		return err
	}

	// Build apply-style object
	applyObj := map[string]interface{}{
		"apiVersion": u.GetAPIVersion(),
		"kind": u.GetKind(),
		"metadata": map[string]interface{}{
			"name": u.GetName(),
			"namespace": u.GetNamespace(),
		},
		"status": sm,
	}
	patchBytes, err := json.Marshal(applyObj)
	if err != nil {
		return err
	}

	resClient := c.dynClient.Resource(operatorGVR).Namespace(u.GetNamespace())

	// Exponential backoff parameters (from controller configuration)
	maxAttempts := c.MaxPatchAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	baseDelay := c.PatchBaseDelay
	if baseDelay <= 0 {
		baseDelay = 200 * time.Millisecond
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		_, lastErr = resClient.Patch(ctx, u.GetName(), types.ApplyPatchType, patchBytes, metav1.PatchOptions{FieldManager: "hlm-operator", Force: boolPtr(true)}, "status")
		if lastErr == nil {
			return nil
		}
		// If not found, return immediately
		if apierrors.IsNotFound(lastErr) {
			return lastErr
		}
		// If conflict or other transient error, retry with backoff
		if apierrors.IsConflict(lastErr) {
			time.Sleep(baseDelay * (1 << attempt))
			continue
		}
		// If server does not support apply (method not allowed / bad request), fall back to merge patch
		if apierrors.IsMethodNotAllowed(lastErr) || apierrors.IsBadRequest(lastErr) {
			// prepare merge patch
			mergeObj := map[string]interface{}{"status": sm}
			mergeBytes, _ := json.Marshal(mergeObj)
			var mergeErr error
			for mAttempt := 0; mAttempt < maxAttempts; mAttempt++ {
				_, mergeErr = resClient.Patch(ctx, u.GetName(), types.MergePatchType, mergeBytes, metav1.PatchOptions{}, "status")
				if mergeErr == nil {
					return nil
				}
				if apierrors.IsConflict(mergeErr) {
					time.Sleep(baseDelay * (1 << mAttempt))
					continue
				}
				return mergeErr
			}
			return fmt.Errorf("merge patch failed after retries: %w", mergeErr)
		}
		// For other errors, treat as transient and retry a few times
		time.Sleep(baseDelay * (1 << attempt))
	}
	if lastErr != nil {
		return fmt.Errorf("apply failed after %d attempts: %w", maxAttempts, lastErr)
	}
	return fmt.Errorf("apply failed after %d attempts", maxAttempts)
}

// createOrUpdateRenderedConfigMap stores the rendered manifest into a new ConfigMap with timestamped name and retains last 7 history entries
func (c *Controller) createOrUpdateRenderedConfigMap(ctx context.Context, u *unstructured.Unstructured, manifest string) (string, error) {
	baseName := u.GetName()
	cmName := fmt.Sprintf("%s-rendered-%d", baseName, time.Now().Unix())
	ns := u.GetNamespace()
	cmClient := c.kubeClient.CoreV1().ConfigMaps(ns)
	owner := metav1.OwnerReference{
		APIVersion: u.GetAPIVersion(),
		Kind: u.GetKind(),
		Name: u.GetName(),
		UID: u.GetUID(),
		Controller: boolPtr(true),
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: cmName,
			Namespace: ns,
			Labels: map[string]string{"hlm-rendered-of": baseName},
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		Data: map[string]string{"manifest": manifest},
	}
	_, err := cmClient.Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	// cleanup older entries, keep last 7
	list, err := cmClient.List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("hlm-rendered-of=%s", baseName)})
	if err != nil {
		return cmName, nil // created successfully, but cleanup failed -> ignore
	}
	// sort by creationTimestamp ascending
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].GetCreationTimestamp().Time.Before(list.Items[j].GetCreationTimestamp().Time)
	})
	if len(list.Items) > 7 {
		toDelete := len(list.Items) - 7
		for i := 0; i < toDelete; i++ {
			old := &list.Items[i]
			_ = cmClient.Delete(ctx, old.GetName(), metav1.DeleteOptions{})
		}
	}
	return cmName, nil
}

func (c *Controller) handle(obj interface{}, eventType string) {
	u := obj.(*unstructured.Unstructured)
	var spec HlmSpec
	// marshal spec map to JSON then unmarshal into typed spec
	if s, ok := u.Object["spec"]; ok {
		sb, err := json.Marshal(s)
		if err == nil {
			_ = json.Unmarshal(sb, &spec)
		}
	}

	releaseName := spec.ReleaseName
	if releaseName == "" {
		releaseName = u.GetName()
	}
	targetNs := spec.TargetNamespace
	if targetNs == "" {
		targetNs = c.namespace
	}

	// create a context for Helm operations
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// prepare a base status
	baseStatus := HlmStatus{}
	baseStatus.LastReconcile = metav1.Now()

	// If not yet marked for deletion, ensure finalizer exists
	if u.GetDeletionTimestamp() == nil {
		_ = c.ensureFinalizer(context.Background(), u)
	}

	// If deletion timestamp is set, handle finalizer workflow and uninstall
	if u.GetDeletionTimestamp() != nil {
		// call uninstall
		res, err := c.helm.UninstallChart(ctx, releaseName, targetNs)
		if err != nil {
			baseStatus.LastAction = "Uninstall"
			baseStatus.LastActionStatus = "Failed"
			baseStatus.LastActionMessage = err.Error()
			_ = c.patchStatusForObject(context.Background(), u, baseStatus)
			c.recorder.Event(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: u.GetName()}}, corev1.EventTypeWarning, "HelmUninstallFailed", err.Error())
			fmt.Printf("Failed to uninstall release %s in %s: %v\n", releaseName, targetNs, err)
			return
		}
		// success: remove finalizer
		baseStatus.LastAction = "Uninstall"
		baseStatus.LastActionStatus = "Succeeded"
		baseStatus.LastActionMessage = fmt.Sprintf("Uninstalled release: %v", res)
		_ = c.patchStatusForObject(context.Background(), u, baseStatus)
		if err := c.removeFinalizer(context.Background(), u); err != nil {
			fmt.Printf("Failed to remove finalizer for %s: %v\n", u.GetName(), err)
		}
		c.recorder.Event(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: u.GetName()}}, corev1.EventTypeNormal, "HelmUninstalled", "Helm release uninstalled")
		fmt.Printf("Uninstalled release %s in namespace %s\n", releaseName, targetNs)
		return
	}

	// For ADD/UPDATE: render template and store in ConfigMap, record CM name to status
	tmpl, terr := c.helm.TemplateChart(ctx, "tpl_vipex_cc-0.1.0.tgz", spec.Values, releaseName, targetNs)
	if terr != nil {
		baseStatus.LastAction = "Template"
		baseStatus.LastActionStatus = "Failed"
		baseStatus.LastActionMessage = terr.Error()
		_ = c.patchStatusForObject(context.Background(), u, baseStatus)
		c.recorder.Event(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: u.GetName()}}, corev1.EventTypeWarning, "HelmTemplateFailed", terr.Error())
		fmt.Printf("Template render failed for %s/%s: %v\n", targetNs, releaseName, terr)
	} else {
		cmName, cmErr := c.createOrUpdateRenderedConfigMap(ctx, u, tmpl)
		if cmErr != nil {
			baseStatus.LastAction = "Template"
			baseStatus.LastActionStatus = "Failed"
			baseStatus.LastActionMessage = cmErr.Error()
			_ = c.patchStatusForObject(context.Background(), u, baseStatus)
			c.recorder.Event(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: u.GetName()}}, corev1.EventTypeWarning, "ConfigMapFailed", cmErr.Error())
			fmt.Printf("Failed to create/update ConfigMap for %s: %v\n", u.GetName(), cmErr)
		} else {
			baseStatus.LastAction = "Template"
			baseStatus.LastActionStatus = "Succeeded"
			baseStatus.RenderedManifestConfigMap = cmName
			baseStatus.LastActionMessage = "Template rendered and stored in ConfigMap"
			_ = c.patchStatusForObject(context.Background(), u, baseStatus)
			c.recorder.Event(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: u.GetName()}}, corev1.EventTypeNormal, "HelmTemplate", "Rendered manifest stored in ConfigMap")
			fmt.Printf("Rendered manifest stored in ConfigMap %s/%s\n", u.GetNamespace(), cmName)
		}
	}

	// execute helm install/upgrade
	res, err := c.helm.InstallOrUpgradeChart(ctx, "tpl_vipex_cc-0.1.0.tgz", releaseName, targetNs, spec.Values)
	if err != nil {
		// attempt rollback on upgrade failure
		baseStatus.LastAction = "InstallOrUpgrade"
		baseStatus.LastActionStatus = "Failed"
		baseStatus.LastActionMessage = err.Error()
		_ = c.patchStatusForObject(context.Background(), u, baseStatus)
		c.recorder.Event(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: u.GetName()}}, corev1.EventTypeWarning, "HelmFailed", err.Error())
		fmt.Printf("Install/upgrade failed for %s/%s: %v\n", targetNs, releaseName, err)

		// attempt rollback
		rbErr := c.helm.RollbackChart(ctx, releaseName, targetNs)
		if rbErr != nil {
			baseStatus.LastAction = "Rollback"
			baseStatus.LastActionStatus = "Failed"
			baseStatus.LastActionMessage = fmt.Sprintf("rollback failed: %v; original error: %v", rbErr, err)
			_ = c.patchStatusForObject(context.Background(), u, baseStatus)
			c.recorder.Event(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: u.GetName()}}, corev1.EventTypeWarning, "HelmRollbackFailed", rbErr.Error())
			fmt.Printf("Rollback failed for %s/%s: %v\n", targetNs, releaseName, rbErr)
			return
		}
		// rollback succeeded
		baseStatus.LastAction = "Rollback"
		baseStatus.LastActionStatus = "Succeeded"
		baseStatus.LastActionMessage = "Rollback succeeded after failed upgrade"
		_ = c.patchStatusForObject(context.Background(), u, baseStatus)
		c.recorder.Event(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: u.GetName()}}, corev1.EventTypeNormal, "HelmRollback", "Rollback succeeded")
		fmt.Printf("Rollback succeeded for %s/%s\n", targetNs, releaseName)
		return
	}

	// success
	baseStatus.LastAction = "InstallOrUpgrade"
	baseStatus.LastActionStatus = "Succeeded"
	baseStatus.LastActionMessage = "Applied chart"
	if res != nil {
		baseStatus.LastRevision = res.Version
	}
	_ = c.patchStatusForObject(context.Background(), u, baseStatus)
	c.recorder.Event(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: u.GetName()}}, corev1.EventTypeNormal, "HelmApplied", "Helm chart applied")
	fmt.Printf("Applied chart for release %s in namespace %s\n", releaseName, targetNs)
}

// ensureFinalizer ensures the finalizer exists on the resource
func (c *Controller) ensureFinalizer(ctx context.Context, u *unstructured.Unstructured) error {
	resClient := c.dynClient.Resource(operatorGVR).Namespace(u.GetNamespace())
	latest, err := resClient.Get(ctx, u.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	finalizers := latest.GetFinalizers()
	for _, f := range finalizers {
		if f == finalizerName {
			return nil
		}
	}
	finalizers = append(finalizers, finalizerName)
	latest.SetFinalizers(finalizers)
	_, err = resClient.Update(ctx, latest, metav1.UpdateOptions{})
	return err
}

// removeFinalizer removes the operator finalizer from the resource
func (c *Controller) removeFinalizer(ctx context.Context, u *unstructured.Unstructured) error {
	resClient := c.dynClient.Resource(operatorGVR).Namespace(u.GetNamespace())
	latest, err := resClient.Get(ctx, u.GetName(), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	finalizers := latest.GetFinalizers()
	newFinalizers := make([]string, 0, len(finalizers))
	for _, f := range finalizers {
		if f == finalizerName {
			continue
		}
		newFinalizers = append(newFinalizers, f)
	}
	latest.SetFinalizers(newFinalizers)
	_, err = resClient.Update(ctx, latest, metav1.UpdateOptions{})
	return err
}

func boolPtr(b bool) *bool { return &b }
