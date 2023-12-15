package cmd

import (
	"github.com/edgeflare/edge/pkg/config"
	"github.com/edgeflare/edge/pkg/logger"
	"github.com/edgeflare/edge/pkg/version"
	"github.com/urfave/cli/v2"
)

// SetupApp sets up the CLI app
func SetupApp() *cli.App {
	app := &cli.App{
		Name:    "edge",
		Usage:   "manage k3s clusters and containerized apps (as helmcharts) anywhere",
		Version: version.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "config file path (default: $HOME/.edge/edge.yaml, if it exists)",
				EnvVars: []string{"EDGE_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "logging level. options: debug, info, warn, error",
				Value:   "info",
				EnvVars: []string{"EDGE_LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:    "kubeconfig",
				Aliases: []string{"k"},
				Usage:   "kubeconfig file. in-cluster configuration (ServiceAccount) is used if not provided",
				EnvVars: []string{"KUBECONFIG"},
			},
			&cli.StringSliceFlag{
				Name:    "extraHelmRepos",
				Aliases: []string{"r"},
				Usage:   "extra repos, in the form 'name=repo_url', can be specified multiple times",
				EnvVars: []string{"EDGE_EXTRAHELMREPOS"},
			},
		},
		Before: func(c *cli.Context) error {
			conf, err := config.Load(c)
			if err != nil {
				return err
			}

			logLevel := conf.LogLevel
			logger.Initialize(logLevel)

			extraHelmRepos := []config.HelmRepository{}
			if conf.ExtraHelmRepos != nil {
				extraHelmRepos = *conf.ExtraHelmRepos
			}
			config.InitCombinedHelmRepositories(config.GetDefaulHelmRepositories(), extraHelmRepos)

			return nil
		},
		Commands: []*cli.Command{
			k3sCommand(),
			kubectlCommand(),
			serverCommand(),
		},
	}

	return app
}
