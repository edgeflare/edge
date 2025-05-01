package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
			// Create context that can be cancelled
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Set up signal handling
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

			// Start services in goroutines
			go runEnvoyControlplaneServer(ctx)
			go serve(ctx, port, zitadelHost)

			// Wait for termination signal
			<-sig
			log.Println("Shutting down...")
			cancel() // This will propagate to all services

			// Give everything time to shut down
			time.Sleep(1 * time.Second)
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

	// envoy controlplane flags
	serveCmd.Flags().UintVarP(&xdsPort, "xds-port", "x", 18000, "xDS management server port")
	serveCmd.Flags().StringVarP(&envoyConfigFile, "envoy-routes", "r", "", "Path to the Envoy configuration file")
	serveCmd.Flags().StringVarP(&envoyNodeID, "envoy-node-id", "n", "envoy-node", "Node ID for Envoy")
	serveCmd.Flags().StringVarP(&envoyCertFile, "envoy-cert", "C", "/etc/envoy/tls.crt", "Path to the TLS certificate file")
	serveCmd.Flags().StringVarP(&envoyKeyFile, "envoy-key", "K", "/etc/envoy/tls.key", "Path to the TLS key file")
	serveCmd.Flags().UintVarP(&envoyHttpPort, "envoy-http-port", "H", 10080, "HTTP port for Envoy")
	serveCmd.Flags().UintVarP(&envoyHttpsPort, "envoy-https-port", "S", 10443, "HTTPS port for Envoy")

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
