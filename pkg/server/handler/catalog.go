package handler

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/edgeflare/edge/pkg/config"
	"github.com/labstack/echo/v4"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/yaml"
)

// Cache is a struct to hold the cached data
type Cache struct {
	lastFetch time.Time
	data      []byte
}

// HelmIndex represents a Helm repository index
type HelmIndex struct {
	Entries map[string][]ChartVersion `yaml:"entries"`
}

// ChartVersion represents a Helm chart version
type ChartVersion struct {
	Version string   `yaml:"version"`
	URLs    []string `yaml:"urls"`
}

var (
	indexCacheMap = make(map[string]Cache)
	cacheDuration = 60 * time.Minute // adjust this as per your requirement
)

// ListHelmRepos returns the list of Helm repositories
func ListHelmRepos(c echo.Context) error {
	return c.JSON(http.StatusOK, config.GetCombinedHelmRepositories())
}

// GetHelmRepoIndex returns the index.yaml for a Helm repository
func GetHelmRepoIndex(c echo.Context) error {
	repoName := c.Param("repoName")
	indexData, err := fetchAndCacheIndex(repoName)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	jsonData, err := yaml.YAMLToJSON(indexData)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert YAML to JSON")
	}

	return c.Blob(http.StatusOK, "application/json", jsonData)
}

// GetHelmRepoChartSpec returns the chart data for a Helm chart
func GetHelmRepoChartSpec(c echo.Context) error {
	repoName := c.Param("repoName")
	chartName := c.Param("chartName")
	version := c.Param("version")

	indexData, err := fetchAndCacheIndex(repoName)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	chartURL, err := parseIndexAndGetChartURL(indexData, chartName, version, getRepoURLByName(repoName))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	chartResp, err := http.Get(chartURL)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}
	defer chartResp.Body.Close()

	chartData, err := io.ReadAll(chartResp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read chart package")
	}

	chart, err := parseChart(chartData)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse chart package")
	}

	jsonData, err := json.Marshal(chart)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert chart to JSON")
	}

	return c.Blob(http.StatusOK, "application/json", jsonData)
}

func parseChart(chartData []byte) (*chart.Chart, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(chartData))
	if err != nil {
		return nil, fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// var chart chart.Chart
	var parsedChart chart.Chart
	chartFound := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar entry: %v", err)
		}

		if header.Typeflag == tar.TypeReg {
			fileData, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("error reading file '%s': %v", header.Name, err)
			}

			switch {
			case strings.HasSuffix(header.Name, "Chart.yaml"):
				err = yaml.Unmarshal(fileData, &parsedChart.Metadata)
				if err != nil {
					return nil, fmt.Errorf("error parsing Chart.yaml: %v", err)
				}
				chartFound = true
			case strings.HasSuffix(header.Name, "values.yaml"):
				jsonData, err := yaml.YAMLToJSON(fileData)
				if err != nil {
					return nil, fmt.Errorf("error converting values.yaml to JSON: %v", err)
				}

				err = json.Unmarshal(jsonData, &parsedChart.Values)
				if err != nil {
					return nil, fmt.Errorf("error parsing values.yaml: %v", err)
				}
				fileName := path.Base(header.Name)
				parsedChart.Files = append(parsedChart.Files, &chart.File{
					Name: fileName,
					Data: fileData,
				})

			case strings.Contains(header.Name, "/templates/"):
				trimmedName := strings.TrimPrefix(header.Name, strings.Split(header.Name, "/templates/")[0]+"/")
				parsedChart.Templates = append(parsedChart.Templates, &chart.File{
					Name: trimmedName,
					Data: fileData,
				})
			case strings.HasSuffix(strings.ToLower(header.Name), "readme.md"):
				fileName := path.Base(header.Name)
				parsedChart.Files = append(parsedChart.Files, &chart.File{
					Name: fileName,
					Data: fileData,
				})
			}
		}
	}

	if !chartFound {
		return nil, fmt.Errorf("file Chart.yaml not found in the chart package")
	}

	return &parsedChart, nil
}

func getRepoURLByName(name string) string {
	for _, repo := range config.GetCombinedHelmRepositories() {
		if repo.Name == name {
			return repo.URL
		}
	}
	return ""
}

func fetchAndCacheIndex(repoName string) ([]byte, error) {
	if cache, ok := indexCacheMap[repoName]; ok && time.Since(cache.lastFetch) < cacheDuration {
		return cache.data, nil
	}

	repoURL := getRepoURLByName(repoName)
	if repoURL == "" {
		return nil, fmt.Errorf("invalid repo name provided")
	}

	resp, err := http.Get(repoURL + "/index.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository index")
	}
	defer resp.Body.Close()

	indexData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read index data")
	}

	indexCacheMap[repoName] = Cache{
		data:      indexData,
		lastFetch: time.Now(),
	}

	return indexData, nil
}

func parseIndexAndGetChartURL(indexData []byte, chartName, version, baseURL string) (string, error) {
	var index HelmIndex
	err := yaml.Unmarshal(indexData, &index)
	if err != nil {
		return "", fmt.Errorf("failed to parse index file: %v", err)
	}

	chartVersions, ok := index.Entries[chartName]
	if !ok {
		return "", fmt.Errorf("chart '%s' not found", chartName)
	}

	for _, chart := range chartVersions {
		if chart.Version == version {
			if len(chart.URLs) > 0 {
				chartURL := chart.URLs[0]
				if !strings.HasPrefix(chartURL, "http://") && !strings.HasPrefix(chartURL, "https://") {
					chartURL = baseURL + "/" + chartURL
				}
				return chartURL, nil
			}
			return "", fmt.Errorf("no URL found for chart '%s' version '%s'", chartName, version)
		}
	}

	return "", fmt.Errorf("version '%s' of chart '%s' not found", version, chartName)
}
