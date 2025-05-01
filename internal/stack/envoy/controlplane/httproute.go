package controlplane

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	corsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	httpv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"sigs.k8s.io/yaml"
)

const (
	defaultHTTPPort      = uint32(80)
	defaultHTTPSPort     = uint32(443)
	defaultNodeID        = "envoy-node"
	connectTimeout       = 5 * time.Second
	maxConcurrentStreams = 100
	initStreamWindowSize = 65536
	initConnWindowSize   = 1048576
)

// Config types
type (
	GatewayConfig struct {
		Routes map[string]HTTPRoute `json:"routes"`
	}

	HTTPRoute struct {
		Name      string          `json:"-"`
		Hostnames []string        `json:"hostnames"`
		Rules     []HTTPRouteRule `json:"rules"`
	}

	HTTPRouteRule struct {
		Matches     []HTTPRouteMatch `json:"matches"`
		BackendRefs []BackendRef     `json:"backendRefs"`
	}

	HTTPRouteMatch struct {
		Path    *PathMatch        `json:"path,omitempty"`
		Method  *string           `json:"method,omitempty"`
		Headers map[string]string `json:"headers,omitempty"`
	}

	PathMatch struct {
		Type  string `json:"type"` // "Exact" or "Prefix"
		Value string `json:"value"`
	}

	BackendRef struct {
		Host  string `json:"host"`
		Port  uint32 `json:"port"`
		HTTP2 bool   `json:"http2,omitempty"`
	}

	// Manager options
	TLSConfig struct {
		CertFile string `json:"certFile"`
		KeyFile  string `json:"keyFile"`
		CAFile   string `json:"caFile"`
	}

	HTTPConfig struct {
		Port uint32 `json:"port"`
	}

	HTTPSConfig struct {
		Port uint32    `json:"port"`
		TLS  TLSConfig `json:"tls"`
	}

	RouteManagerOptions struct {
		NodeID string
		HTTP   HTTPConfig  `json:"http"`
		HTTPS  HTTPSConfig `json:"https"`
	}

	// RouteManager handles route configuration and updates
	RouteManager struct {
		mu            sync.RWMutex
		config        GatewayConfig
		snapshotCache cachev3.SnapshotCache
		version       int64
		nodeID        string
		ctx           context.Context
		cancel        context.CancelFunc
		httpPort      uint32
		httpsPort     uint32
		certPath      string
		keyPath       string
		enableHTTPS   bool
	}
)

// NewRouteManager creates a new route manager
func NewRouteManager(cache cachev3.SnapshotCache, opts RouteManagerOptions) *RouteManager {
	ctx, cancel := context.WithCancel(context.Background())

	httpPort := defaultHTTPPort
	if opts.HTTP.Port > 0 {
		httpPort = opts.HTTP.Port
	}

	httpsPort := defaultHTTPSPort
	if opts.HTTPS.Port > 0 {
		httpsPort = opts.HTTPS.Port
	}

	nodeID := opts.NodeID
	if nodeID == "" {
		nodeID = defaultNodeID
	}

	enableHTTPS := opts.HTTPS.Port != 0 && opts.HTTPS.TLS.CertFile != "" && opts.HTTPS.TLS.KeyFile != ""

	return &RouteManager{
		config:        GatewayConfig{Routes: make(map[string]HTTPRoute)},
		snapshotCache: cache,
		version:       1,
		nodeID:        nodeID,
		ctx:           ctx,
		cancel:        cancel,
		httpPort:      httpPort,
		httpsPort:     httpsPort,
		certPath:      opts.HTTPS.TLS.CertFile,
		keyPath:       opts.HTTPS.TLS.KeyFile,
		enableHTTPS:   enableHTTPS,
	}
}

// Close cleans up resources
func (rm *RouteManager) Close() {
	if rm.cancel != nil {
		rm.cancel()
	}
}

// updateSnapshot creates and applies a new xDS snapshot
func (rm *RouteManager) updateSnapshot() error {
	routes := make([]HTTPRoute, 0, len(rm.config.Routes))
	for _, route := range rm.config.Routes {
		routes = append(routes, route)
	}

	version := fmt.Sprintf("%d", atomic.AddInt64(&rm.version, 1))
	log.Printf("Updating snapshot v%s with %d routes", version, len(routes))

	var (
		snapshot *cachev3.Snapshot
		err      error
	)

	if rm.enableHTTPS {
		snapshot, err = rm.generateSnapshotWithHTTPS(routes, version)
	} else {
		snapshot, err = rm.generateSnapshot(routes, version)
	}

	if err != nil {
		return fmt.Errorf("failed to generate snapshot: %w", err)
	}

	ctx, cancel := context.WithTimeout(rm.ctx, 5*time.Second)
	defer cancel()
	return rm.snapshotCache.SetSnapshot(ctx, rm.nodeID, snapshot)
}

// LoadFromYAML loads configuration from a YAML file
func (rm *RouteManager) LoadFromYAML(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read YAML file: %w", err)
	}

	var config GatewayConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	for name, route := range config.Routes {
		route.Name = name
		config.Routes[name] = route
	}

	rm.config = config
	return rm.updateSnapshot()
}

// SaveToYAML saves configuration to a YAML file
func (rm *RouteManager) SaveToYAML(filename string) error {
	rm.mu.RLock()
	data, err := yaml.Marshal(rm.config)
	rm.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return os.WriteFile(filename, data, 0644)
}

// Route management methods
func (rm *RouteManager) GetAllRoutes() map[string]HTTPRoute {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	routes := make(map[string]HTTPRoute, len(rm.config.Routes))
	for k, v := range rm.config.Routes {
		routes[k] = v
	}
	return routes
}

func (rm *RouteManager) GetRoute(name string) (HTTPRoute, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	route, exists := rm.config.Routes[name]
	return route, exists
}

func (rm *RouteManager) CreateRoute(name string, route HTTPRoute) error {
	if name == "" {
		return fmt.Errorf("route name cannot be empty")
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.config.Routes[name]; exists {
		return fmt.Errorf("route %s already exists", name)
	}

	route.Name = name
	rm.config.Routes[name] = route
	return rm.updateSnapshot()
}

func (rm *RouteManager) UpdateRoute(name string, route HTTPRoute) error {
	if name == "" {
		return fmt.Errorf("route name cannot be empty")
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.config.Routes[name]; !exists {
		return fmt.Errorf("route %s does not exist", name)
	}

	route.Name = name
	rm.config.Routes[name] = route
	return rm.updateSnapshot()
}

func (rm *RouteManager) DeleteRoute(name string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.config.Routes[name]; !exists {
		return fmt.Errorf("route %s does not exist", name)
	}

	delete(rm.config.Routes, name)
	return rm.updateSnapshot()
}

// Helper constructor methods
func NewHTTPRoute(name string, hostnames []string) HTTPRoute {
	return HTTPRoute{Name: name, Hostnames: hostnames}
}

func (route *HTTPRoute) AddRule(rule HTTPRouteRule) *HTTPRoute {
	route.Rules = append(route.Rules, rule)
	return route
}

func (rule *HTTPRouteRule) AddPathPrefix(prefix string) *HTTPRouteRule {
	rule.Matches = append(rule.Matches, HTTPRouteMatch{
		Path: &PathMatch{Type: "Prefix", Value: prefix},
	})
	return rule
}

func (rule *HTTPRouteRule) AddPathExact(path string) *HTTPRouteRule {
	rule.Matches = append(rule.Matches, HTTPRouteMatch{
		Path: &PathMatch{Type: "Exact", Value: path},
	})
	return rule
}

func (rule *HTTPRouteRule) AddCatchAllRoute() *HTTPRouteRule {
	rule.Matches = append(rule.Matches, HTTPRouteMatch{
		Path: &PathMatch{Type: "Prefix", Value: "/"},
	})
	return rule
}

// Envoy configuration generation
func (rm *RouteManager) generateSnapshot(routes []HTTPRoute, version string) (*cachev3.Snapshot, error) {
	backends := collectBackendRefs(routes)

	clusters := make([]types.Resource, 0, len(backends))
	for _, backend := range backends {
		clusters = append(clusters, makeCluster(backend))
	}

	routeConfig := makeRouteConfig(routes)
	httpListener := makeHTTPListener(routeConfig.Name, rm.httpPort)
	if httpListener == nil {
		return nil, fmt.Errorf("failed to create HTTP listener")
	}

	return cachev3.NewSnapshot(version,
		map[resource.Type][]types.Resource{
			resource.ClusterType:  clusters,
			resource.RouteType:    {routeConfig},
			resource.ListenerType: {httpListener},
		},
	)
}

func (rm *RouteManager) generateSnapshotWithHTTPS(routes []HTTPRoute, version string) (*cachev3.Snapshot, error) {
	backends := collectBackendRefs(routes)

	clusters := make([]types.Resource, 0, len(backends))
	for _, backend := range backends {
		clusters = append(clusters, makeCluster(backend))
	}

	routeConfig := makeRouteConfig(routes)
	httpListener := makeHTTPListener(routeConfig.Name, rm.httpPort)
	if httpListener == nil {
		return nil, fmt.Errorf("failed to create HTTP listener")
	}

	httpsListener := makeHTTPSListener(routeConfig.Name, rm.httpsPort, rm.certPath, rm.keyPath)
	if httpsListener == nil {
		return nil, fmt.Errorf("failed to create HTTPS listener")
	}

	return cachev3.NewSnapshot(version,
		map[resource.Type][]types.Resource{
			resource.ClusterType:  clusters,
			resource.RouteType:    {routeConfig},
			resource.ListenerType: {httpListener, httpsListener},
		},
	)
}

// collectBackendRefs collects unique backend references
func collectBackendRefs(routes []HTTPRoute) []BackendRef {
	backendMap := make(map[string]BackendRef)

	for _, route := range routes {
		for _, rule := range route.Rules {
			for _, backendRef := range rule.BackendRefs {
				key := fmt.Sprintf("%s:%d", backendRef.Host, backendRef.Port)
				backendMap[key] = backendRef
			}
		}
	}

	backends := make([]BackendRef, 0, len(backendMap))
	for _, backend := range backendMap {
		backends = append(backends, backend)
	}

	return backends
}

// makeClusterName generates a consistent cluster name
func makeClusterName(host string, port uint32) string {
	return fmt.Sprintf("cluster_%s_%v", host, port)
}

// Envoy resource creation helpers
func makeConfigSource() *corev3.ConfigSource {
	return &corev3.ConfigSource{
		ResourceApiVersion: resource.DefaultAPIVersion,
		ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
			ApiConfigSource: &corev3.ApiConfigSource{
				TransportApiVersion:       resource.DefaultAPIVersion,
				ApiType:                   corev3.ApiConfigSource_GRPC,
				SetNodeOnFirstMessageOnly: true,
				GrpcServices: []*corev3.GrpcService{{
					TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{ClusterName: "xds_cluster"},
					},
				}},
			},
		},
	}
}

func makeEndpoint(clusterName string, backendRef BackendRef) *endpointv3.ClusterLoadAssignment {
	return &endpointv3.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpointv3.LocalityLbEndpoints{{
			LbEndpoints: []*endpointv3.LbEndpoint{{
				HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
					Endpoint: &endpointv3.Endpoint{
						Address: &corev3.Address{
							Address: &corev3.Address_SocketAddress{
								SocketAddress: &corev3.SocketAddress{
									Protocol: corev3.SocketAddress_TCP,
									Address:  backendRef.Host,
									PortSpecifier: &corev3.SocketAddress_PortValue{
										PortValue: backendRef.Port,
									},
								},
							},
						},
					},
				},
			}},
		}},
	}
}

func makeCluster(backendRef BackendRef) *clusterv3.Cluster {
	clusterName := makeClusterName(backendRef.Host, backendRef.Port)
	cluster := &clusterv3.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       durationpb.New(connectTimeout),
		ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_LOGICAL_DNS},
		LbPolicy:             clusterv3.Cluster_ROUND_ROBIN,
		LoadAssignment:       makeEndpoint(clusterName, backendRef),
		DnsLookupFamily:      clusterv3.Cluster_V4_ONLY,
	}

	// Add HTTP protocol options
	if err := addHttpProtocolOptions(cluster, backendRef.HTTP2); err != nil {
		log.Printf("Failed to add HTTP protocol options to cluster %s: %v", clusterName, err)
	}

	return cluster
}

func addHttpProtocolOptions(cluster *clusterv3.Cluster, preferHttp2 bool) error {
	httpOpts := &httpv3.HttpProtocolOptions{
		CommonHttpProtocolOptions: &corev3.HttpProtocolOptions{
			IdleTimeout: durationpb.New(30 * time.Second),
		},
		UpstreamHttpProtocolOptions: &corev3.UpstreamHttpProtocolOptions{
			AutoSni: true,
		},
	}

	hasTLS := cluster.TransportSocket != nil && cluster.TransportSocket.Name == "envoy.transport_sockets.tls"

	http2Options := &corev3.Http2ProtocolOptions{
		MaxConcurrentStreams:        &wrapperspb.UInt32Value{Value: maxConcurrentStreams},
		InitialStreamWindowSize:     &wrapperspb.UInt32Value{Value: initStreamWindowSize},
		InitialConnectionWindowSize: &wrapperspb.UInt32Value{Value: initConnWindowSize},
	}

	httpOptions := &corev3.Http1ProtocolOptions{
		HeaderKeyFormat: &corev3.Http1ProtocolOptions_HeaderKeyFormat{
			HeaderFormat: &corev3.Http1ProtocolOptions_HeaderKeyFormat_ProperCaseWords_{
				ProperCaseWords: &corev3.Http1ProtocolOptions_HeaderKeyFormat_ProperCaseWords{},
			},
		},
	}

	if hasTLS {
		httpOpts.UpstreamProtocolOptions = &httpv3.HttpProtocolOptions_AutoConfig{
			AutoConfig: &httpv3.HttpProtocolOptions_AutoHttpConfig{
				Http2ProtocolOptions: http2Options,
				HttpProtocolOptions:  httpOptions,
			},
		}
	} else if preferHttp2 {
		httpOpts.UpstreamProtocolOptions = &httpv3.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{
					Http2ProtocolOptions: &corev3.Http2ProtocolOptions{
						MaxConcurrentStreams:        &wrapperspb.UInt32Value{Value: maxConcurrentStreams},
						InitialStreamWindowSize:     &wrapperspb.UInt32Value{Value: initStreamWindowSize},
						InitialConnectionWindowSize: &wrapperspb.UInt32Value{Value: initConnWindowSize},
						AllowConnect:                true,
					},
				},
			},
		}
	} else {
		httpOpts.UpstreamProtocolOptions = &httpv3.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_HttpProtocolOptions{
					HttpProtocolOptions: httpOptions,
				},
			},
		}
	}

	any, err := anypb.New(httpOpts)
	if err != nil {
		return err
	}

	if cluster.TypedExtensionProtocolOptions == nil {
		cluster.TypedExtensionProtocolOptions = make(map[string]*anypb.Any)
	}

	cluster.TypedExtensionProtocolOptions["envoy.extensions.upstreams.http.v3.HttpProtocolOptions"] = any
	return nil
}

func makeHTTPListener(routeConfigName string, port uint32) *listenerv3.Listener {
	manager, err := makeHTTPConnectionManager(routeConfigName)
	if err != nil {
		log.Printf("Failed to create HTTP connection manager: %v", err)
		return nil
	}

	pbst, err := anypb.New(manager)
	if err != nil {
		log.Printf("Failed to create HTTP connection manager config: %v", err)
		return nil
	}

	return &listenerv3.Listener{
		Name: "http_listener",
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Protocol: corev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
		FilterChains: []*listenerv3.FilterChain{{
			Filters: []*listenerv3.Filter{{
				Name: "envoy.filters.network.http_connection_manager",
				ConfigType: &listenerv3.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}
}

func makeHTTPSListener(routeConfigName string, port uint32, certPath, keyPath string) *listenerv3.Listener {
	manager, err := makeHTTPConnectionManager(routeConfigName)
	if err != nil {
		log.Printf("Failed to create HTTP connection manager for HTTPS: %v", err)
		return nil
	}

	pbst, err := anypb.New(manager)
	if err != nil {
		log.Printf("Failed to create HTTP connection manager config for HTTPS: %v", err)
		return nil
	}

	tlsConfig := &tlsv3.DownstreamTlsContext{
		CommonTlsContext: &tlsv3.CommonTlsContext{
			AlpnProtocols: []string{"h2", "http/1.1"},
			TlsCertificates: []*tlsv3.TlsCertificate{
				{
					CertificateChain: &corev3.DataSource{
						Specifier: &corev3.DataSource_Filename{Filename: certPath},
					},
					PrivateKey: &corev3.DataSource{
						Specifier: &corev3.DataSource_Filename{Filename: keyPath},
					},
				},
			},
		},
	}

	tlsAny, err := anypb.New(tlsConfig)
	if err != nil {
		log.Printf("Failed to create TLS config: %v", err)
		return nil
	}

	return &listenerv3.Listener{
		Name: "https_listener",
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Protocol: corev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
		FilterChains: []*listenerv3.FilterChain{{
			Filters: []*listenerv3.Filter{{
				Name: "envoy.filters.network.http_connection_manager",
				ConfigType: &listenerv3.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
			TransportSocket: &corev3.TransportSocket{
				Name: "envoy.transport_sockets.tls",
				ConfigType: &corev3.TransportSocket_TypedConfig{
					TypedConfig: tlsAny,
				},
			},
		}},
	}
}

func makeHTTPConnectionManager(routeConfigName string) (*hcmv3.HttpConnectionManager, error) {
	corsConfig, err := anypb.New(&corsv3.Cors{})
	if err != nil {
		return nil, err
	}

	routerConfig, err := anypb.New(&routerv3.Router{})
	if err != nil {
		return nil, err
	}

	manager := &hcmv3.HttpConnectionManager{
		CodecType:  hcmv3.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcmv3.HttpConnectionManager_Rds{
			Rds: &hcmv3.Rds{
				ConfigSource:    makeConfigSource(),
				RouteConfigName: routeConfigName,
			},
		},
		HttpFilters: []*hcmv3.HttpFilter{
			{
				Name:       "envoy.filters.http.cors",
				ConfigType: &hcmv3.HttpFilter_TypedConfig{TypedConfig: corsConfig},
			},
			{
				Name:       "envoy.filters.http.router",
				ConfigType: &hcmv3.HttpFilter_TypedConfig{TypedConfig: routerConfig},
			},
		},
		UpgradeConfigs: []*hcmv3.HttpConnectionManager_UpgradeConfig{{
			UpgradeType: "websocket",
		}},
		Http2ProtocolOptions: &corev3.Http2ProtocolOptions{
			MaxConcurrentStreams:        &wrapperspb.UInt32Value{Value: maxConcurrentStreams},
			InitialStreamWindowSize:     &wrapperspb.UInt32Value{Value: initStreamWindowSize},
			InitialConnectionWindowSize: &wrapperspb.UInt32Value{Value: initConnWindowSize},
		},
		HttpProtocolOptions: &corev3.Http1ProtocolOptions{
			HeaderKeyFormat: &corev3.Http1ProtocolOptions_HeaderKeyFormat{
				HeaderFormat: &corev3.Http1ProtocolOptions_HeaderKeyFormat_ProperCaseWords_{
					ProperCaseWords: &corev3.Http1ProtocolOptions_HeaderKeyFormat_ProperCaseWords{},
				},
			},
		},
	}

	return manager, nil
}

// Modified makeVirtualHost function to handle empty matches
func makeVirtualHost(httpRoute HTTPRoute) *routev3.VirtualHost {
	var routes []*routev3.Route
	var hostRewrite string

	// Use first hostname for host rewrite if available
	if len(httpRoute.Hostnames) > 0 {
		hostRewrite = httpRoute.Hostnames[0]
	}

	for _, rule := range httpRoute.Rules {
		if len(rule.BackendRefs) == 0 {
			continue
		}

		// Handle the case when no matches are specified
		if len(rule.Matches) == 0 {
			// Add a default catch-all route with '/' prefix
			defaultMatch := HTTPRouteMatch{
				Path: &PathMatch{Type: "Prefix", Value: "/"},
			}

			route := &routev3.Route{
				Match: makeRouteMatch(defaultMatch),
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: makeClusterName(rule.BackendRefs[0].Host, rule.BackendRefs[0].Port),
						},
					},
				},
			}

			if hostRewrite != "" {
				route.GetRoute().HostRewriteSpecifier = &routev3.RouteAction_HostRewriteLiteral{
					HostRewriteLiteral: hostRewrite,
				}
			}

			routes = append(routes, route)
			continue
		}

		// Process matches normally if they exist
		for _, match := range rule.Matches {
			route := &routev3.Route{
				Match: makeRouteMatch(match),
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: makeClusterName(rule.BackendRefs[0].Host, rule.BackendRefs[0].Port),
						},
					},
				},
			}

			if hostRewrite != "" {
				route.GetRoute().HostRewriteSpecifier = &routev3.RouteAction_HostRewriteLiteral{
					HostRewriteLiteral: hostRewrite,
				}
			}

			routes = append(routes, route)
		}
	}

	domains := make([]string, 0, len(httpRoute.Hostnames))
	for _, host := range httpRoute.Hostnames {
		domains = append(domains, host)
	}

	// Default to wildcard if no domains specified
	if len(domains) == 0 {
		domains = []string{"*"}
	}

	return &routev3.VirtualHost{
		Name:    fmt.Sprintf("vh_%s", httpRoute.Name),
		Domains: domains,
		Routes:  routes,
	}
}

func makeRouteMatch(httpMatch HTTPRouteMatch) *routev3.RouteMatch {
	match := &routev3.RouteMatch{}

	if httpMatch.Path != nil {
		switch httpMatch.Path.Type {
		case "Exact":
			match.PathSpecifier = &routev3.RouteMatch_Path{Path: httpMatch.Path.Value}
		default: // Default to prefix
			match.PathSpecifier = &routev3.RouteMatch_Prefix{Prefix: httpMatch.Path.Value}
		}
	}

	if httpMatch.Method != nil {
		match.Headers = append(match.Headers, &routev3.HeaderMatcher{
			Name: ":method",
			HeaderMatchSpecifier: &routev3.HeaderMatcher_ExactMatch{
				ExactMatch: *httpMatch.Method,
			},
		})
	}

	for name, value := range httpMatch.Headers {
		match.Headers = append(match.Headers, &routev3.HeaderMatcher{
			Name: name,
			HeaderMatchSpecifier: &routev3.HeaderMatcher_ExactMatch{
				ExactMatch: value,
			},
		})
	}

	return match
}

func makeRouteConfig(httpRoutes []HTTPRoute) *routev3.RouteConfiguration {
	virtualHosts := make([]*routev3.VirtualHost, 0, len(httpRoutes))
	for _, route := range httpRoutes {
		virtualHosts = append(virtualHosts, makeVirtualHost(route))
	}
	return &routev3.RouteConfiguration{
		Name:         "httproutes",
		VirtualHosts: virtualHosts,
	}
}
