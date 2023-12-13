package cmd

import (
	"os"

	"github.com/edgeflare/edge/pkg/ssh"
	"github.com/urfave/cli/v2"
)

// ConsoleWriter writes SSH command outputs to the console
type ConsoleWriter struct{}

func (c ConsoleWriter) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p) // Write to standard output
}

func createSSHClientFromContext(c *cli.Context) (*ssh.Client, error) {
	return ssh.NewSSHClient(
		c.String("host"),
		c.String("user"),
		c.String("password"),
		c.String("keyfile"),
		c.Int("port"),
	)
}

var sshFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "host",
		Aliases:  []string{"H"},
		Usage:    "host IP or domain name",
		Required: true,
	},
	&cli.StringFlag{
		Name:     "user",
		Aliases:  []string{"u"},
		Usage:    "user name",
		Required: true,
	},
	&cli.StringFlag{
		Name:    "password",
		Aliases: []string{"p"},
		Usage:   "password for ssh authentication",
	},
	&cli.StringFlag{
		Name:    "keyfile",
		Aliases: []string{"k"},
		Usage:   "path to ssh private key file",
	},
	&cli.IntFlag{
		Name:    "port",
		Aliases: []string{"P"},
		Usage:   "SSH port",
		Value:   22,
	},
	&cli.StringSliceFlag{
		Name:    "env",
		Aliases: []string{"e"},
		Usage:   "environment variables in the form KEY=VALUE",
	},
}
