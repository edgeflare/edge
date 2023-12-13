package server

import (
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
	url, err := url.Parse(target)
	if err != nil {
		zap.L().Error("error parsing target URL", zap.String("target", target), zap.Error(err))
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(url)

	return func(c echo.Context) error {
		// Obtain the original request
		req := c.Request()

		// Rewrite the path
		originalPath := req.URL.Path
		newPath := rewritePath(originalPath)
		req.URL.Path = newPath
		// req.Header.Add("Authorization", "Bearer ...")

		// Serve the request using the reverse proxy
		proxy.ServeHTTP(c.Response(), req)
		return nil
	}
}

// rewritePath rewrites the path for the proxy request
func rewritePath(originalPath string) string {
	// rewriting logic that trims "/api/gateways" prefix from the path
	return strings.TrimPrefix(originalPath, "/api/gateways")
}
