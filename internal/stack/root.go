package main

import (
	"fmt"
	"os"

	"github.com/edgeflare/edge/internal/stack/envoy"
	"github.com/spf13/cobra"
)

func Main() {
	var port int
	var zitadelHost string
	var healthEndpoint string

	// Root command
	rootCmd := &cobra.Command{
		Use:   "health-service",
		Short: "Health check service for your containerized applications",
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
	serveCmd.Flags().IntVarP(&port, "port", "p", 8081, "Port to serve the health API on")
	serveCmd.Flags().StringVarP(&zitadelHost, "zitadel-host", "z", fmt.Sprintf("iam.%s", os.Getenv("EDGE_DOMAIN_ROOT")), "Zitadel host for health checks")

	// Add flags to check command
	checkCmd.Flags().StringVarP(&healthEndpoint, "endpoint", "e", "http://localhost:8081/healthz", "Health endpoint to check")

	// Add commands to root
	rootCmd.AddCommand(serveCmd, checkCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
