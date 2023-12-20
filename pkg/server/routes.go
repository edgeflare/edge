package server

import (
	"os"

	"github.com/edgeflare/edge/pkg/config"
	"github.com/edgeflare/edge/pkg/server/handler"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RegisterHandlers registers all handlers for the API
func RegisterHandlers(e *echo.Echo, c *config.Config) {
	// Set config
	conf = c

	// Static routes
	e.GET("/config.json", GetHTTPConfig)

	// API routes
	api := e.Group("/api/")
	{
		// Auth middleware
		if conf.HTTP.Auth.Type == "jwt" {
			config := echojwt.Config{
				KeyFunc: getKey,
			}
			api.Use(echojwt.WithConfig(config))
			api.Use(verifyClientID)
		} else if conf.HTTP.Auth.Type == "basic" {
			api.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
				if username == conf.HTTP.Auth.Basic.Username && password == conf.HTTP.Auth.Basic.Password {
					return true, nil
				}
				return false, nil
			}))
		}

		// kubernetes resources
		api.GET("api-resources", handler.ListAPIResources)
		api.GET("namespaces", handler.GetNamespaces)
		api.GET("namespaces/:namespace/:resourceType/:resourceName", handler.GetResources) // specific resource in specific namespace
		api.GET("namespaces/:namespace/:resourceType", handler.GetResources)               // in specific namespace
		api.POST("namespaces/:namespace/:resourceType/:resourceName", handler.ApplyResource)
		api.DELETE("namespaces/:namespace/:resourceType/:resourceName", handler.DeleteResource)

		// k3s clusters
		cluster := api.Group("clusters")
		{
			cluster.GET("", handler.ListClusters)
			cluster.GET("/versions", handler.K3sStableVersions)
			cluster.POST("/install", handler.K3sInstall)
			cluster.POST("/:clusterId/join", handler.K3sJoin)
			cluster.GET("/:clusterId/nodes", handler.ListNodes)
			cluster.POST("/:clusterId/nodes/:nodeId", handler.K3sUninstall)
		}

		// helm.cattle.io/v1
		cattle := api.Group("cattle/")
		{
			cattle.GET("helmcharts", handler.ListCattleHelmCharts)                       // in all namespaces
			cattle.GET("namespaces/:namespace/helmcharts", handler.ListCattleHelmCharts) // in specific namespace
			cattle.POST("namespaces/:namespace/helmcharts", handler.CreateOrUpdateCattleHelmChart)
			cattle.GET("namespaces/:namespace/helmcharts/:releaseName", handler.GetCattleHelmChart)
			cattle.DELETE("namespaces/:namespace/helmcharts/:releaseName", handler.DeleteCattleHelmChart)
		}

		// helmreleases
		api.GET("helmcharts", handler.ListHelmChartReleases) // all namespaces
		api.GET("namespaces/:namespace/helmcharts", handler.ListHelmChartReleases)
		api.GET("namespaces/:namespace/helmcharts/:releaseName", handler.GetHelmChartRelease)
		api.GET("namespaces/:namespace/helmcharts/:releaseName/workloads", handler.GetHelmChartReleaseWithWorkloads)

		// charts catalog
		repo := api.Group("catalog/helm/repos")
		{
			repo.GET("", handler.ListHelmRepos)
			repo.GET("/:repoName/charts", handler.GetHelmRepoIndex)
			repo.GET("/:repoName/charts/:chartName/:version", handler.GetHelmRepoChartSpec)
		}

		// gateway routes
		edgeGatewayURL := os.Getenv("EDGE_GATEWAY_URL")
		if edgeGatewayURL == "" {
			edgeGatewayURL = "https://gw.edgeflare.io"
		}
		api.Any("gateways/*", gatewayProxy(edgeGatewayURL))
	}
}
