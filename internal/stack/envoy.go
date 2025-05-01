package main

import (
	"context"
	"fmt"
	"log"

	"github.com/edgeflare/edge/internal/stack/envoy"
	"github.com/edgeflare/edge/internal/stack/envoy/controlplane"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

var (
	xdsPort         uint
	envoyNodeID     string
	envoyConfigFile string
	envoyCertFile   string
	envoyKeyFile    string
	envoyHttpPort   uint
	envoyHttpsPort  uint
)

// runEnvoyControlplaneServer starts the Envoy xDS server with the provided context
func runEnvoyControlplaneServer(ctx context.Context) error {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	routeManager := controlplane.NewRouteManager(snapshotCache, controlplane.RouteManagerOptions{
		NodeID: envoyNodeID,
		HTTP: controlplane.HTTPConfig{
			Port: uint32(envoyHttpPort),
		},
		HTTPS: controlplane.HTTPSConfig{
			Port: uint32(envoyHttpsPort),
			TLS: controlplane.TLSConfig{
				CertFile: envoyCertFile,
				KeyFile:  envoyKeyFile,
			},
		},
	})
	defer routeManager.Close()

	// Load configuration from YAML file
	if err := routeManager.LoadFromYAML(envoyConfigFile); err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	// Create the xDS server
	xdsServer := server.NewServer(ctx, snapshotCache, nil)

	log.Printf("Starting xDS server on port %d", xdsPort)
	return envoy.RunServer(ctx, xdsServer, xdsPort)
}
