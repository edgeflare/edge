package handler

import (
	"net/http"

	"github.com/edgeflare/edge/pkg/kube"
	"github.com/labstack/echo/v4"
)

// ListHelmChartReleases returns the list of Helm chart releases
func ListHelmChartReleases(c echo.Context) error {
	namespace := c.Param("namespace")

	release, err := kube.ListHelmChartReleases(namespace)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, release)
}

// GetHelmChartRelease returns the Helm chart release
func GetHelmChartRelease(c echo.Context) error {
	namespace := c.Param("namespace")
	releaseName := c.Param("releaseName")

	resources, err := kube.GetHelmChartRelease(namespace, releaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, resources)
}

// GetHelmChartReleaseWithWorkloads returns the Helm chart release with workloads
func GetHelmChartReleaseWithWorkloads(c echo.Context) error {
	namespace := c.Param("namespace")
	releaseName := c.Param("releaseName")

	resources, err := kube.GetHelmChartReleaseWithWorkloads(namespace, releaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, resources)
}

func GetChartReleaseRevisions(c echo.Context) error {
	namespace := c.Param("namespace")
	releaseName := c.Param("releaseName")

	releases, err := kube.GetChartReleaseHistory(namespace, releaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, releases)
}
