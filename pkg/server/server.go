package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/edgeflare/edge/pkg/config"
	"github.com/labstack/echo/v4"
	"github.com/lestrrat-go/jwx/jwk"
	"go.uber.org/zap"
)

var (
	jwkCachedKeySet jwk.Set
	jwkLastFetched  time.Time
	jwkCacheTTL     = time.Hour // Cache for 1 hour. Adjust as needed.
	conf            *config.Config
)

// HTTPAuthConfigResponse is the response for HTTP authentication config
type HTTPAuthConfigResponse struct {
	JWT  *config.HTTPJWTAuthConfig `json:"jwt"`
	Type string                    `json:"type"`
}

// HTTPConfigResponse is the response for HTTP config
type HTTPConfigResponse struct {
	Auth *HTTPAuthConfigResponse `json:"auth"`
}

// GetHTTPConfig returns the HTTP config for the UI
func GetHTTPConfig(c echo.Context) error {
	resp := &HTTPConfigResponse{
		Auth: &HTTPAuthConfigResponse{
			Type: conf.HTTP.Auth.Type,
			JWT:  conf.HTTP.Auth.JWT,
		},
	}
	return c.JSON(http.StatusOK, resp)
}

// gatewayProxy creates a reverse proxy handler
func gatewayProxy(target string) echo.HandlerFunc {
	targetURL, err := url.Parse(target)
	if err != nil {
		zap.L().Error("error parsing target URL", zap.String("target", target), zap.Error(err))
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Configure the Director function
	proxy.Director = func(req *http.Request) {
		// Update the scheme and host for the proxy request
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host

		// Update the Host header to the target host
		req.Host = targetURL.Host

		// Trim the '/api/gateways' prefix from the path
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/gateways")
	}

	// Configure the proxy transport
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			// InsecureSkipVerify: true,
			ServerName: targetURL.Hostname(),
		},
	}

	return func(c echo.Context) error {
		// Serve the request using the reverse proxy
		proxy.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
