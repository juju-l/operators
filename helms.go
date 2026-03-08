package main

import (
	"context"
	"fmt"
	"io"
	"os"

	chartloader "helm.sh/helm/v4/pkg/chart/loader"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
)

// HelmClient wraps helm actions needed by the operator
type HelmClient struct {
	cfg *action.Configuration
	settings *cli.EnvSettings
}

// NewHelmClientInCluster creates a Helm client configured for in-cluster use
func NewHelmClientInCluster(namespace string, out io.Writer) (*HelmClient, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)

	cfg := new(action.Configuration)
	if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		fmt.Fprintf(out, format, v...)
	}); err != nil {
		return nil, err
	}
	return &HelmClient{cfg: cfg, settings: settings}, nil
}

// InstallOrUpgradeChart installs a chart archive (.tgz) into target namespace with given release name
func (h *HelmClient) InstallOrUpgradeChart(ctx context.Context, chartArchive string, releaseName string, namespace string, vals map[string]interface{}) (*action.Release, error) {
	r, err := chartloader.Load(chartArchive)
	if err != nil {
		return nil, err
	}

	// check if release exists
	s := action.NewStatus(h.cfg)
	if _, err := s.Run(releaseName); err == nil {
		// release exists, do upgrade
		u := action.NewUpgrade(h.cfg)
		u.Namespace = namespace
		u.ResetValues = false
		res, err := u.Run(r, vals)
		return res, err
	}

	// release not found -> install
	i := action.NewInstall(h.cfg)
	i.Namespace = namespace
	i.ReleaseName = releaseName
	res, err := i.Run(r, vals)
	return res, err
}

// UninstallChart uninstalls a release by name in the given namespace
func (h *HelmClient) UninstallChart(ctx context.Context, releaseName string, namespace string) (*action.UninstallReleaseResponse, error) {
	u := action.NewUninstall(h.cfg)
	u.Namespace = namespace
	res, err := u.Run(releaseName)
	return res, err
}

// TemplateChart renders a chart archive with given values and returns the rendered manifest
// releaseName and namespace are used when rendering; if releaseName is empty Helm will generate one
func (h *HelmClient) TemplateChart(ctx context.Context, chartArchive string, vals map[string]interface{}, releaseName string, namespace string) (string, error) {
	ch, err := chartloader.Load(chartArchive)
	if err != nil {
		return "", err
	}
	i := action.NewInstall(h.cfg)
	i.DryRun = true
	i.ClientOnly = true
	if releaseName != "" {
		i.ReleaseName = releaseName
	}
	if namespace != "" {
		i.Namespace = namespace
	}

	res, err := i.Run(ch, vals)
	if err != nil {
		return "", err
	}
	return res.Manifest, nil
}

// ReleaseStatus checks whether a release exists and returns the release if present
func (h *HelmClient) ReleaseStatus(ctx context.Context, releaseName string) (*action.Release, error) {
	s := action.NewStatus(h.cfg)
	res, err := s.Run(releaseName)
	return res, err
}

// RollbackChart attempts to rollback a release to its previous revision
func (h *HelmClient) RollbackChart(ctx context.Context, releaseName string, namespace string) error {
	rb := action.NewRollback(h.cfg)
	rb.Namespace = namespace
	// default behaviour: wait for rollback to complete
	rb.Wait = true
	if err := rb.Run(releaseName); err != nil {
		return err
	}
	return nil
}
