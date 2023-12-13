package k3s

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"go.uber.org/zap"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
}

// GetLatestK3sVersions returns the latest K3s versions
func GetLatestK3sVersions() ([]string, error) {
	url := "https://api.github.com/repos/k3s-io/k3s/releases"
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	var versions []string
	for _, release := range releases {
		if !release.Prerelease {
			versions = append(versions, release.TagName)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		baseVersion1, err1 := semver.NewVersion(strings.Split(versions[i], "+")[0])
		if err1 != nil {
			zap.L().Error("Invalid semantic version", zap.String("version", versions[i]), zap.Error(err1))
			return false
		}

		baseVersion2, err2 := semver.NewVersion(strings.Split(versions[j], "+")[0])
		if err2 != nil {
			zap.L().Error("Invalid semantic version", zap.String("version", versions[j]), zap.Error(err2))
			return false
		}

		if baseVersion1.Equal(baseVersion2) {
			// Compare the numeric part after '+k3s'
			return extractK3sNumber(versions[i]) > extractK3sNumber(versions[j])
		}
		return baseVersion1.GreaterThan(baseVersion2)
	})

	return versions, nil
}

// extractK3sNumber extracts the numeric value after '+k3s'
func extractK3sNumber(version string) int {
	parts := strings.Split(version, "+k3s")
	if len(parts) < 2 {
		return 0
	}
	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return num
}
