package main

import (
	"fmt"
	"os"

	"github.com/edgeflare/edge/internal/stack/envoy"
	"github.com/spf13/cobra"
)

var (
	port           = 8081
	zitadelHost    = fmt.Sprintf("iam.%s", os.Getenv("EDGE_DOMAIN_ROOT"))
	healthEndpoint = fmt.Sprintf("http://localhost:%v/healthz", port)
	addons         []string
)

func Main() {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "edge",
		Short: "PostgreSQL backend using OIDC IdP, Envoy, PGo, NATS, ...",
	}

	// Serve command
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the health check API server",
		Run: func(cmd *cobra.Command, args []string) {
			// checkZitadelAdminSA()
			go envoy.RunControlplaneServer()
			serve(port, zitadelHost)
		},
	}

	// Health check command
	checkCmd := &cobra.Command{
		Use:   "healthz",
		Short: "Check health status of services",
		Run: func(cmd *cobra.Command, args []string) {
			if err := checkHealth(healthEndpoint); err != nil {
				fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add flags to serve command
	serveCmd.Flags().IntVarP(&port, "port", "p", port, "Port to serve the health API on")
	serveCmd.Flags().StringVarP(&zitadelHost, "zitadel-host", "z", fmt.Sprintf("iam.%s", os.Getenv("EDGE_DOMAIN_ROOT")), "Zitadel host for health checks")
	serveCmd.Flags().StringSliceVarP(&addons, "configure-addons", "c", []string{}, "extra comma-separated add-ons to configure eg emqx,nats, ... to use IdP for authZ")

	// Add flags to check command
	checkCmd.Flags().StringVarP(&healthEndpoint, "endpoint", "e", healthEndpoint, "Health endpoint to check")

	// Add commands to root
	rootCmd.AddCommand(serveCmd, checkCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
