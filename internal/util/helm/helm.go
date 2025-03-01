package helm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"sigs.k8s.io/yaml"
)

type ReleaseSpec struct {
	Name string
	// OCI URL (e.g., "registry-1.docker.io/bitnamicharts/postgresql:16.4.9")
	ChartURL      string
	Namespace     string
	ValuesContent string
}

type Client struct {
	env      *cli.EnvSettings
	registry *registry.Client
}

// NewClient returns a new helm registry client
func NewClient() (*Client, error) {
	env := cli.New()
	reg, err := registry.NewClient(
		registry.ClientOptDebug(env.Debug),
		registry.ClientOptCredentialsFile(env.RegistryConfig),
	)
	if err != nil {
		return nil, fmt.Errorf("registry client creation failed: %w", err)
	}

	return &Client{env: env, registry: reg}, nil
}

func (c *Client) newActionConfig(namespace string) (*action.Configuration, error) {
	cfg := new(action.Configuration)
	if err := cfg.Init(c.env.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"),
		func(f string, v ...any) {
			if !strings.HasSuffix(f, "\n") {
				f += "\n"
			}
			fmt.Printf(f, v...)
		}); err != nil {
		return nil, fmt.Errorf("action config init failed: %w", err)
	}
	cfg.RegistryClient = c.registry
	return cfg, nil
}

// Install installs a chart release, or upgrades if it already exists
func (c *Client) Install(ctx context.Context, rel ReleaseSpec) (*release.Release, error) {
	cfg, err := c.newActionConfig(rel.Namespace)
	if err != nil {
		return nil, err
	}

	chartRef := rel.ChartURL
	if !strings.HasPrefix(chartRef, "oci://") {
		chartRef = "oci://" + chartRef
	}

	history, err := c.getReleaseHistory(rel.Name, rel.Namespace)
	exists := err == nil && len(history) > 0

	var chartPath string
	if exists {
		upgrade := action.NewUpgrade(cfg)
		upgrade.Namespace = rel.Namespace
		chartPath, err = upgrade.ChartPathOptions.LocateChart(chartRef, c.env)
	} else {
		install := action.NewInstall(cfg)
		install.ReleaseName = rel.Name
		install.Namespace = rel.Namespace
		chartPath, err = install.ChartPathOptions.LocateChart(chartRef, c.env)
	}
	if err != nil {
		return nil, fmt.Errorf("chart location failed: %w", err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("chart loading failed: %w", err)
	}

	values, err := parseYAMLValues(rel.ValuesContent)
	if err != nil {
		return nil, fmt.Errorf("values parsing failed: %w", err)
	}

	if exists {
		upgrade := action.NewUpgrade(cfg)
		upgrade.Namespace = rel.Namespace
		return upgrade.Run(rel.Name, chart, values)
	}

	install := action.NewInstall(cfg)
	install.ReleaseName = rel.Name
	install.Namespace = rel.Namespace
	return install.Run(chart, values)
}

func (c *Client) Uninstall(ctx context.Context, name, namespace string) error {
	cfg, err := c.newActionConfig(namespace)
	if err != nil {
		return err
	}

	_, err = action.NewUninstall(cfg).Run(name)
	return err
}

func (c *Client) ListReleases(ctx context.Context, namespace string) ([]string, error) {
	cfg, err := c.newActionConfig(namespace)
	if err != nil {
		return nil, err
	}

	releases, err := action.NewList(cfg).Run()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(releases))
	for i, r := range releases {
		names[i] = r.Name
	}
	return names, nil
}

func (c *Client) getReleaseHistory(name, namespace string) ([]*release.Release, error) {
	cfg, err := c.newActionConfig(namespace)
	if err != nil {
		return nil, err
	}

	hist := action.NewHistory(cfg)
	hist.Max = 1
	return hist.Run(name)
}

func parseYAMLValues(content string) (map[string]any, error) {
	if content == "" {
		return make(map[string]any), nil
	}

	values := make(map[string]any)
	if err := yaml.Unmarshal([]byte(content), &values); err != nil {
		return nil, fmt.Errorf("YAML parsing failed: %w", err)
	}
	return values, nil
}
