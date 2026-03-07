package main

import (
	"bytes"
	"context"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/client-go/rest"
)

// HelmOperator 封装 Helm 操作
type HelmOperator struct {
	chartPath string
}

// NewHelmOperator 创建实例
func NewHelmOperator(chartPath string) *HelmOperator {
	return &HelmOperator{chartPath: chartPath}
}

// restConfigGetter 适配 client-go rest.Config
type restConfigGetter struct {
	rc *rest.Config
}

func (r *restConfigGetter) ToRESTConfig() (*rest.Config, error) {
	return r.rc, nil
}

func (r *restConfigGetter) ToDiscoveryClient() (rest.DiscoveryInterface, error) {
	return nil, nil
}

func (r *restConfigGetter) ToRawKubeConfigLoader() cli.EnvSettings {
	return cli.EnvSettings{}
}

// getActionConfig 初始化 Helm action config
func (ho *HelmOperator) getActionConfig(ns string, rc *rest.Config) (*action.Configuration, error) {
	env := cli.New()
	env.RESTClientGetter = &restConfigGetter{rc}

	cfg := new(action.Configuration)
	sto := storage.Init(driver.NewSecrets(rc, ns))

	cfg.RESTClientGetter = env.RESTClientGetter
	cfg.Releases = sto
	return cfg, nil
}

// InstallOrUpgrade 安装或升级 Release
func (ho *HelmOperator) InstallOrUpgrade(ctx context.Context, spec *HlmSpec, rc *rest.Config) (*release.Release, error) {
	cfg, err := ho.getActionConfig(spec.Namespace, rc)
	if err != nil {
		return nil, err
	}

	ch, err := loader.Load(ho.chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	v, err := (&values.Options{}).MergeValues(nil, bytes.NewBufferString(spec.ValuesYAML))
	if err != nil {
		return nil, fmt.Errorf("parse values: %w", err)
	}

	hist := action.NewHistory(cfg)
	hist.Max = 1
	_, err = hist.Run(spec.ReleaseName)

	if err != nil {
		inst := action.NewInstall(cfg)
		inst.ReleaseName = spec.ReleaseName
		inst.Namespace = spec.Namespace
		inst.CreateNamespace = true
		return inst.Run(ch, v)
	}

	up := action.NewUpgrade(cfg)
	up.Namespace = spec.Namespace
	return up.Run(spec.ReleaseName, ch, v)
}

// Uninstall 卸载 Release
func (ho *HelmOperator) Uninstall(ctx context.Context, releaseName, ns string, rc *rest.Config) error {
	cfg, err := ho.getActionConfig(ns, rc)
	if err != nil {
		return err
	}
	_, err = action.NewUninstall(cfg).Run(releaseName)
	return err
}
