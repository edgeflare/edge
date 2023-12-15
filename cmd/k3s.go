package cmd

import (
	"fmt"
	"time"

	"github.com/edgeflare/edge/pkg/k3s"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func k3sCommand() *cli.Command {
	return &cli.Command{
		Name:    "k3s",
		Aliases: []string{"cluster", "c"},
		Usage:   "manage k3s clusters",
		Subcommands: []*cli.Command{
			{
				Name:    "install",
				Aliases: []string{"i"},
				Usage:   "Install k3s on a remote host",
				Flags:   append(sshFlags, k3sInstallFlags...),
				Action: func(c *cli.Context) error {
					sshClient, err := createSSHClientFromContext(c)
					if err != nil {
						return fmt.Errorf("error creating SSH client: %w", err)
					}
					k3sService := k3s.NewK3sService(sshClient)

					_, err = k3sService.InstallK3s(
						&ConsoleWriter{},
						c.Bool("cluster"),
						c.String("tls-san"),
						// c.String("node-external-ip"),
						c.String("k3s-args"),
						c.String("version"),
					)
					if err != nil {
						return err
					}

					if !c.Bool("no-copy-kubeconfig") {
						if err := k3sService.DownloadK3sKubeconfig(); err != nil {
							zap.L().Error("error downloading kubeconfig", zap.Error(err))
							return err
						}
					}

					return nil
				},
			},
			{
				Name:    "join",
				Aliases: []string{"j"},
				Usage:   "Join a k3s cluster",
				Flags:   append(sshFlags, k3sJoinFlags...),
				Action: func(c *cli.Context) error {
					sshClient, err := createSSHClientFromContext(c)
					if err != nil {
						return fmt.Errorf("error creating SSH client: %w", err)
					}
					k3sService := k3s.NewK3sService(sshClient)

					nodeID, err := k3sService.JoinK3s(
						&ConsoleWriter{},
						c.String("server"),
						c.Bool("master"),
						c.String("token"),
					)
					if err != nil {
						return err
					}
					fmt.Println(nodeID)

					return nil
				},
			},
			{
				Name:    "uninstall",
				Aliases: []string{"destroy", "d"},
				Usage:   "Uninstall k3s on a remote host",
				Flags:   append(sshFlags, k3sUninstallFlags...),
				Action: func(c *cli.Context) error {
					sshClient, err := createSSHClientFromContext(c)
					if err != nil {
						return fmt.Errorf("error creating SSH client: %w", err)
					}
					k3sService := k3s.NewK3sService(sshClient)

					_, err = k3sService.UninstallK3s(
						&ConsoleWriter{},
						c.Bool("agent"),
					)
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:    "update",
				Aliases: []string{"u"},
				Usage:   "Update k3s on a remote host",
				Flags:   append(sshFlags, k3sUpdateFlags...),
				Action: func(c *cli.Context) error {
					fmt.Println("Not implemented yet")
					return nil
				},
			},
			{
				Name:    "copy-kubeconfig",
				Aliases: []string{"cpk"},
				Usage:   "Copy kubeconfig from remote host",
				Flags:   sshFlags,
				Action: func(c *cli.Context) error {
					sshClient, err := createSSHClientFromContext(c)
					if err != nil {
						return fmt.Errorf("error creating SSH client: %w", err)
					}
					k3sService := k3s.NewK3sService(sshClient)

					if err := k3sService.DownloadK3sKubeconfig(); err != nil {
						zap.L().Error("error downloading kubeconfig", zap.Error(err))
						return err
					}

					return nil
				},
			},
			{
				Name:    "ls",
				Aliases: []string{"l"},
				Usage:   "List k3s clusters",
				Action: func(c *cli.Context) error {
					k3sService := k3s.NewK3sService(nil)
					clusters, err := k3sService.ListClusters()
					if err != nil {
						fmt.Println(err)
						return err
					}

					// Print table header
					fmt.Printf("%-10s\t%-10s\t%-10s\t%-10s\t%-10s\t%-10s\n", "ID", "Status", "Version", "Is HA", "APIserver", "CreatedAt")
					// Print each cluster's information
					for _, cluster := range clusters {
						fmt.Printf("%-10s\t%-10s\t%-10s\t%-10v\t%-10v\t%-10v\n", cluster.ID, cluster.Status, cluster.Version, cluster.IsHA, cluster.Apiserver, cluster.CreatedAt.Format(time.RFC3339))
					}
					return nil
				},
			},
			{
				Name:    "nodes",
				Aliases: []string{"n"},
				Usage:   "List nodes in a k3s cluster",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "clusterid",
						Aliases:  []string{"c"},
						Usage:    "Specify the Cluster ID to list its nodes",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					k3sService := k3s.NewK3sService(nil)
					nodes, err := k3sService.ListNodesByCluster(c.String("clusterid"))
					if err != nil {
						fmt.Println(err)
					}

					// Print table header
					fmt.Printf("%-10s\t%-15s\t%-10s\t%-10s\t%-10s\n", "Node ID", "IP", "Role", "Status", "CreatedAt")
					// Print each node's information
					for i := range nodes {
						fmt.Printf("%-10s\t%-15s\t%-10s\t%-10s\t%-10s\n", nodes[i].ID, nodes[i].IP, nodes[i].Role, nodes[i].Status, nodes[i].CreatedAt.Format(time.RFC3339))
					}
					return nil
				},
			},
		},
	}
}

var k3sInstallFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "tls-san",
		Aliases: []string{"s"},
		Usage:   "Add additional hostnames or IPv4/IPv6 addresses as Subject Alternative Names on the server TLS cert",
	},
	&cli.BoolFlag{
		Name:    "cluster",
		Aliases: []string{"c"},
		Usage:   "Initialize a new cluster using embedded Etcd",
		Value:   false,
		EnvVars: []string{"EDGE_K3S_CLUSTER"},
	},
	// &cli.StringFlag{
	// 	Name:    "node-external-ip",
	// 	Aliases: []string{"ip"},
	// 	Usage:   "IP address of node to use for external communication",
	// },
	&cli.StringFlag{
		Name:  "k3s-args",
		Usage: "Additional arguments to pass to k3s installer",
	},
	&cli.StringFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Install a specific version. Leave empty for the latest stable version",
		Value:   "",
	},
	&cli.BoolFlag{
		Name:    "no-copy-kubeconfig",
		Aliases: []string{"nck"},
		Usage:   "Skip copying kubeconfig from the remote host after installation",
	},
}

var k3sJoinFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "server",
		Aliases:  []string{"s"},
		Usage:    "The server to join",
		Required: true,
	},
	&cli.StringFlag{
		Name:    "token",
		Aliases: []string{"t"},
		Usage:   "The token to join. If not provided, downloaded from --server",
		// Required: true,
	},
	&cli.BoolFlag{
		Name:    "master",
		Aliases: []string{"m"},
		Usage:   "Whether the newly joining node is a server ie control-plane node",
		Value:   false,
	},
}

var k3sUninstallFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "agent",
		Aliases: []string{"a"},
		Usage:   "Uninstall an agent node",
	},
}

var k3sUpdateFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Update to a specific version",
	},
}
