package config

// HelmRepository represents a Helm chart repository.
type HelmRepository struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	// Username string `json:"username,omitempty"` // not used yet
	// Password string `json:"password,omitempty"`
}

var combinedHelmRepositories []HelmRepository

// GetCombinedHelmRepositories returns the combined list of default and extra repositories.
func GetCombinedHelmRepositories() []HelmRepository {
	return combinedHelmRepositories
}

// GetDefaulHelmRepositories returns the default list of repositories.
func GetDefaulHelmRepositories() []HelmRepository {
	return defaultHelmRepositories
}

// InitCombinedHelmRepositories initializes the combined list of default and extra repositories.
func InitCombinedHelmRepositories(defaultRepos, extraRepos []HelmRepository) {
	combinedHelmRepositories = mergeRepositories(defaultRepos, extraRepos)
}

var defaultHelmRepositories = []HelmRepository{
	{
		Name: "bitnami",
		URL:  "https://charts.bitnami.com/bitnami",
	},
	{
		Name: "edgeflare",
		URL:  "https://helm.edgeflare.io",
	},
	{
		Name: "grafana",
		URL:  "https://grafana.github.io/helm-charts",
	},
	{
		Name: "elastic",
		URL:  "https://helm.elastic.co",
	},
	{
		Name: "jetstack",
		URL:  "https://charts.jetstack.io",
	},
	{
		Name: "influxdata",
		URL:  "https://helm.influxdata.com",
	},
	{
		Name: "timescale",
		URL:  "https://charts.timescale.com",
	},
	{
		Name: "emqx",
		URL:  "https://repos.emqx.io/charts",
	},
	{
		Name: "jfrog",
		URL:  "https://charts.jfrog.io",
	},
	{
		Name: "prometheus-community",
		URL:  "https://prometheus-community.github.io/helm-charts",
	},
	{
		Name: "kubernetes-dashboard",
		URL:  "https://kubernetes.github.io/dashboard",
	},
}

// MergeRepositories merges default and extra repositories, avoiding duplicates.
func mergeRepositories(defaultRepos, extraRepos []HelmRepository) []HelmRepository {
	// Use a map to track repositories by their name or URL for quick lookups
	repoMap := make(map[string]bool)

	// First, add all default repositories to the merged list and map
	merged := make([]HelmRepository, len(defaultRepos))
	copy(merged, defaultRepos)
	for _, repo := range defaultRepos {
		repoMap[repo.Name] = true
	}

	// Then, add extra repositories that are not already in the map
	for _, extraRepo := range extraRepos {
		if _, exists := repoMap[extraRepo.Name]; !exists {
			merged = append(merged, extraRepo)
			repoMap[extraRepo.Name] = true
		}
	}

	return merged
}
