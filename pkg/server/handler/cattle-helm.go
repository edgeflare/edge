package handler

import (
	"net/http"

	"github.com/edgeflare/edge/pkg/kube"
	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// ListCattleHelmCharts lists all CattleHelmCharts in all namespaces
func ListCattleHelmCharts(c echo.Context) error {
	namespace := c.Param("namespace")

	helmChartList, err := kube.ListCattleHelmCharts(namespace)
	if err != nil {
		zap.L().Error("Failed to list HelmCharts", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, helmChartList)
}

// GetCattleHelmChart gets a CattleHelmChart in a namespace
func GetCattleHelmChart(c echo.Context) error {
	namespace := c.Param("namespace")
	name := c.Param("releaseName")

	helmChart, err := kube.GetCattleHelmChart(namespace, name)
	if err != nil {
		zap.L().Error("Failed to get HelmChart", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, helmChart)
}

// CreateOrUpdateCattleHelmChart creates or updates a CattleHelmChart in a namespace
func CreateOrUpdateCattleHelmChart(c echo.Context) error {
	namespace := c.Param("namespace")

	newHelmChart := new(helmv1.HelmChart)
	if err := c.Bind(newHelmChart); err != nil {
		zap.L().Error("Failed to bind HelmChart", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	helmChart, err := kube.CreateOrUpdateCattleHelmChart(*newHelmChart, namespace)
	if err != nil {
		zap.L().Error("Failed to create or update HelmChart", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusCreated, helmChart)
}

// DeleteCattleHelmChart deletes a CattleHelmChart in a namespace
func DeleteCattleHelmChart(c echo.Context) error {
	namespace := c.Param("namespace")
	name := c.Param("releaseName")

	err := kube.DeleteCattleHelmChart(namespace, name)
	if err != nil {
		zap.L().Error("Failed to delete HelmChart", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, "OK")
}
