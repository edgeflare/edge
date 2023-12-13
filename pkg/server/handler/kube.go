package handler

import (
	"net/http"

	"github.com/edgeflare/edge/pkg/kube"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// GetNamespaces returns the list of namespaces
func GetNamespaces(c echo.Context) error {
	namespaces, err := kube.GetNamespaces()
	if err != nil {
		zap.L().Error("Failed to get namespaces", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, namespaces)
}
