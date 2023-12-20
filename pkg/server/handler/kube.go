package handler

import (
	"io"
	"net/http"

	"github.com/edgeflare/edge/pkg/kube"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// GetNamespaces handles the request to get a list of Kubernetes namespaces.
func GetNamespaces(c echo.Context) error {
	namespaces, err := kube.GetNamespaces()
	if err != nil {
		return handleError(c, err, "Failed to get namespaces")
	}
	return c.JSON(http.StatusOK, namespaces)
}

// ListApiResources handles the request to list API resources in the Kubernetes cluster.
func ListAPIResources(c echo.Context) error {
	kc, err := kube.NewClient()
	if err != nil {
		return handleError(c, err, "Failed to create kube client")
	}

	apiResources, err := kc.ListAPIResources()
	if err != nil {
		return handleError(c, err, "Failed to get API resources")
	}
	return c.JSON(http.StatusOK, apiResources)
}

// ApplyResource handles the request to apply a Kubernetes resource.
func ApplyResource(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body")
	}

	kc, err := kube.NewClient()
	if err != nil {
		return handleError(c, err, "Failed to create kube client")
	}

	if err = kc.ApplyResource(body); err != nil {
		return handleError(c, err, "Failed to apply resource")
	}

	return c.JSON(http.StatusCreated, string(body))
}

// GetResources handles the request to retrieve Kubernetes resources.
func GetResources(c echo.Context) error {
	resourceType := c.Param("resourceType")
	resourceName := c.Param("resourceName")
	namespace := c.QueryParam("namespace")

	if namespace == "all" {
		namespace = ""
	}

	kc, err := kube.NewClient()
	if err != nil {
		return handleError(c, err, "Failed to create kube client")
	}

	resources, err := kc.GetResources(resourceType, resourceName, namespace)
	if err != nil {
		return handleError(c, err, "Failed to get resources")
	}

	if resourceName != "" {
		return c.JSON(http.StatusOK, resources[0].Object)
	}

	return c.JSON(http.StatusOK, resources)
}

// DeleteResource handles the request to delete a Kubernetes resource.
func DeleteResource(c echo.Context) error {
	resourceType := c.Param("resourceType")
	resourceName := c.Param("resourceName")
	namespace := c.QueryParam("namespace")

	if namespace == "all" {
		namespace = ""
	}

	kc, err := kube.NewClient()
	if err != nil {
		return handleError(c, err, "Failed to create kube client")
	}

	if err = kc.DeleteResource(resourceType, resourceName, namespace, nil); err != nil {
		return handleError(c, err, "Failed to delete resource")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Resource deleted successfully",
	})
}

// handleError logs the error and sends a JSON response with the appropriate HTTP status code.
func handleError(c echo.Context, err error, message string) error {
	zap.L().Error(message, zap.Error(err))
	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error": err.Error(),
	})
}
