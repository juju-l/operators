package main

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helmv1 "your/api/group/api/v1"
)

type HelmReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *HelmReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	obj := &helmv1.Helm{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if obj.Status.Phase == "" {
		if err := InstallHelmChart(ctx, r.Client, obj); err != nil {
			obj.Status.Phase = "Failed"
			obj.Status.Message = err.Error()
			_ = r.Status().Update(ctx, obj)
			return ctrl.Result{}, err
		}

		obj.Status.Phase = "Ready"
		obj.Status.Message = "Installed"
		if err := r.Status().Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *HelmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1.Helm{}).
		Complete(r)
}