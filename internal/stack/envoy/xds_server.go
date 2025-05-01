package envoy

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	grpcKeepaliveTime        = 30 * time.Second
	grpcKeepaliveTimeout     = 5 * time.Second
	grpcKeepaliveMinTime     = 30 * time.Second
	grpcMaxConcurrentStreams = 1000000
)

func registerServer(grpcServer *grpc.Server, server server.Server) {
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
}

// RunServer starts an xDS server at the given port and handles graceful shutdown.
func RunServer(ctx context.Context, srv server.Server, port uint) error {
	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    grpcKeepaliveTime,
			Timeout: grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	}

	// Create a new gRPC server
	grpcServer := grpc.NewServer(grpcOptions...)

	// Listen on the specified port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", port, err)
	}

	// Register the xDS server implementations
	registerServer(grpcServer, srv)

	log.Printf("xDS server listening on %d\n", port)

	// Channel to capture server errors
	serverErr := make(chan error, 1)

	// Start the server
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Println("xDS server shutting down gracefully...")
		// Stop the server gracefully
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()

		// Allow a timeout for graceful shutdown
		shutdownTimeout := 5 * time.Second
		select {
		case <-stopped:
			log.Println("xDS server shutdown complete")
		case <-time.After(shutdownTimeout):
			log.Println("xDS server graceful shutdown timed out, forcing stop")
			grpcServer.Stop()
		}
		return nil
	}
}

// try utilizing test.Callbacks or implement a custom Callbacks struct
//
/*
var (
	l      controlplane.Logger
	port   uint
	nodeID string
)

func init() {
	l = controlplane.Logger{}

	flag.BoolVar(&l.Debug, "debug", false, "Enable xDS server debug logging")

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 18001, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "localhost", "Node ID")
}

func RunControlplaneServer() {
	flag.Parse()

	// Create a cache
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	// Create the snapshot that we'll serve to Envoy
	snapshot, _ := controlplane.GenerateSnapshot(nil, "1")
	if err := snapshot.Consistent(); err != nil {
		l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
		os.Exit(1)
	}
	l.Debugf("will serve snapshot %+v", snapshot)

	// Add the snapshot to the cache
	if err := cache.SetSnapshot(context.Background(), nodeID, snapshot); err != nil {
		l.Errorf("snapshot error %q for %+v", err, snapshot)
		os.Exit(1)
	}

	// Run the xDS server
	ctx := context.Background()
	cb := &test.Callbacks{Debug: l.Debug}
	srv := server.NewServer(ctx, cache, cb)
	controlplane.RunServer(srv, port)
}
*/
