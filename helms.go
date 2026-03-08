package main

import (
	"context"
	"fmt"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart"
	"helm.sh/helm/v4/pkg/chart/loader"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/storage"
	"helm.sh/helm/v4/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helmv1 "your/api/group/api/v1"
)

func getActionConfig(ctx context.Context, c client.Client, namespace string) (*action.Configuration, error) {
	restConfig := ctrl.GetConfigOrDie()
	helmClient := cli.New()
	helmClient.SetKubeConfig(restConfig)

	driver, err := driver.NewSecrets(driver.SecretsConfig{
		Client:    c,
		Namespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret driver: %w", err)
	}

	cfg := &action.Configuration{
		RESTClientGetter: helmClient,
		Storage:          storage.Init(driver),
	}
	return cfg, nil
}

func InstallHelmChart(ctx context.Context, c client.Client, obj *helmv1.Helm) error {
	log := log.FromContext(ctx)
	cfg, err := getActionConfig(ctx, c, obj.Namespace)
	if err != nil {
		return fmt.Errorf("action config failed: %w", err)
	}

	chrt, err := loader.Load(obj.Spec.Chart)
	if err != nil {
		return fmt.Errorf("load chart failed: %w", err)
	}

	install := action.NewInstall(cfg)
	install.ReleaseName = obj.Spec.ReleaseName
	install.Namespace = obj.Spec.Namespace

	rel, err := install.Run(ctx, chrt, nil)
	if err != nil {
		return fmt.Errorf("install run failed: %w", err)
	}

	log.Info("Chart installed", "release", rel.Name)
	return nil
}

func UpgradeHelmChart(ctx context.Context, c client.Client, obj *helmv1.Helm) error {
	cfg, err := getActionConfig(ctx, c, obj.Namespace)
	if err != nil {
		return fmt.Errorf("action config failed: %w", err)
	}

	chrt, err := loader.Load(obj.Spec.Chart)
	if err != nil {
		return fmt.Errorf("load chart failed: %w", err)
	}

	upgrade := action.NewUpgrade(cfg)
	upgrade.Namespace = obj.Spec.Namespace

	rel, err := upgrade.Run(ctx, obj.Spec.ReleaseName, chrt, nil)
	if err != nil {
		return fmt.Errorf("upgrade run failed: %w", err)
	}

	log.FromContext(ctx).Info("Chart upgraded", "release", rel.Name)
	return nil
}