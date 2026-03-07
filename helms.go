package helms

import (
	"bytes"
	"context"
	"fmt"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart/loader"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/cli/values"
	"helm.sh/helm/v4/pkg/release"
	"helm.sh/helm/v4/pkg/storage"
	"helm.sh/helm/v4/pkg/storage/driver"
	"k8s.io/client-go/rest"
)

// HelmOperator 封装 Helm 操作（Helm v4 兼容）
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

// InstallOrUpgrade 安装或升级 Release（Helm v4 兼容，传入 ctx）
func (ho *HelmOperator) InstallOrUpgrade(ctx context.Context, spec *HlmSpec, rc *rest.Config) (*release.Release, error) {
	cfg, err := ho.getActionConfig(spec.Namespace, rc)
	if err != nil {
		return nil, fmt.Errorf("init action config: %w", err)
	}

	ch, err := loader.Load(ho.chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	valsOpt := &values.Options{}
	vals, err := valsOpt.MergeValues(nil, bytes.NewBufferString(spec.ValuesYAML))
	if err != nil {
		return nil, fmt.Errorf("parse valuesYAML: %w", err)
	}

	histClient := action.NewHistory(cfg)
	histClient.Max = 1
	_, err = histClient.Run(ctx, spec.ReleaseName)

	var rel *release.Release
	if err != nil {
		installClient := action.NewInstall(cfg)
		installClient.ReleaseName = spec.ReleaseName
		installClient.Namespace = spec.Namespace
		installClient.CreateNamespace = true
		rel, err = installClient.Run(ctx, ch, vals)
	} else {
		upgradeClient := action.NewUpgrade(cfg)
		upgradeClient.Namespace = spec.Namespace
		rel, err = upgradeClient.Run(ctx, spec.ReleaseName, ch, vals)
	}

	if err != nil {
		return nil, fmt.Errorf("helm install/upgrade: %w", err)
	}
	return rel, nil
}

// Uninstall 卸载 Release（Helm v4 兼容，传入 ctx）
func (ho *HelmOperator) Uninstall(ctx context.Context, releaseName, ns string, rc *rest.Config) error {
	cfg, err := ho.getActionConfig(ns, rc)
	if err != nil {
		return fmt.Errorf("init action config: %w", err)
	}

	uninstallClient := action.NewUninstall(cfg)
	_, err = uninstallClient.Run(ctx, releaseName)
	if err != nil {
		return fmt.Errorf("helm uninstall: %w", err)
	}
	return nil
}
