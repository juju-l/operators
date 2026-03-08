package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	chartloader "helm.sh/helm/v4/pkg/chart/loader"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart"
	commonchart "helm.sh/helm/v4/pkg/chart/common"
	util "helm.sh/helm/v4/pkg/chart/common/util"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/engine"
)

// HelmClient wraps helm actions needed by the operator
type HelmClient struct {
	cfg      *action.Configuration
	settings *cli.EnvSettings
}

// NewHelmClientInCluster creates a Helm client configured for in-cluster use
func NewHelmClientInCluster(namespace string, out io.Writer) (*HelmClient, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)

	cfg := new(action.Configuration)
	// Init takes RESTClientGetter, namespace, and driver
	if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER")); err != nil {
		return nil, err
	}
	return &HelmClient{cfg: cfg, settings: settings}, nil
}

// InstallOrUpgradeChart installs a chart archive (.tgz) into target namespace with given release name
// returns the release revision on success (0 if unknown)
func (h *HelmClient) InstallOrUpgradeChart(ctx context.Context, chartArchive string, releaseName string, namespace string, vals map[string]interface{}) (int, error) {
	ch, err := chartloader.Load(chartArchive)
	if err != nil {
		return 0, err
	}

	// check if release exists
	s := action.NewStatus(h.cfg)
	if _, err := s.Run(releaseName); err == nil {
		// release exists, do upgrade
		u := action.NewUpgrade(h.cfg)
		_, err := u.Run(releaseName, ch, vals)
		if err != nil {
			return 0, err
		}
		// Helm release object not reliably typed here; return unknown revision
		return 0, nil
	}

	// release not found -> install
	i := action.NewInstall(h.cfg)
	i.ReleaseName = releaseName
	_, err = i.Run(ch, vals)
	if err != nil {
		return 0, err
	}
	return 0, nil
}

// UninstallChart uninstalls a release by name in the given namespace
// returns a short textual result on success
func (h *HelmClient) UninstallChart(ctx context.Context, releaseName string, namespace string) (string, error) {
	u := action.NewUninstall(h.cfg)
	res, err := u.Run(releaseName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%+v", res), nil
}

// TemplateChart renders a chart archive with given values and returns the rendered manifest
// releaseName and namespace are used when rendering; if releaseName is empty Helm will generate one
// This implementation uses Helm's rendering engine via chart/common util and engine.
func (h *HelmClient) TemplateChart(ctx context.Context, chartArchive string, vals map[string]interface{}, releaseName string, namespace string) (string, error) {
	ch, err := chartloader.Load(chartArchive)
	if err != nil {
		return "", err
	}
	// convert to values for rendering
	relOpts := commonchart.ReleaseOptions{
		Name:      releaseName,
		Namespace: namespace,
		Revision:  1,
		IsInstall: true,
	}
	valsToRender, err := util.ToRenderValues(ch.(chart.Charter), vals, relOpts, commonchart.DefaultCapabilities)
	if err != nil {
		return "", err
	}
	eng := engine.Engine{}
	manifests, err := eng.Render(ch.(chart.Charter), valsToRender)
	if err != nil {
		return "", err
	}
	// concatenate in sorted key order for determinism
	keys := make([]string, 0, len(manifests))
	for k := range manifests {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, k := range keys {
		m := manifests[k]
		if strings.TrimSpace(m) == "" {
			continue
		}
		out.WriteString(m)
		out.WriteString("\n---\n")
	}
	return out.String(), nil
}

// RollbackChart attempts to rollback a release to its previous revision
func (h *HelmClient) RollbackChart(ctx context.Context, releaseName string, namespace string) error {
	rb := action.NewRollback(h.cfg)
	if err := rb.Run(releaseName); err != nil {
		return err
	}
	return nil
}
