package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// WorkloadStatus represents the status of a Kubernetes workload.
type WorkloadStatus struct {
	Status interface{} `json:"status"`
	Name   string      `json:"name"`
	Kind   string      `json:"kind"`
}

// ReleaseWithWorkloadStatus represents a Helm chart release with the status of its workloads.
type ReleaseWithWorkloadStatus struct {
	*release.Release
	Manifest  interface{}      `json:"manifest"`
	Workloads []WorkloadStatus `json:"workloads"`
}

// GetHelmChartRelease returns a Helm chart release with the given namespace and release name.
func GetHelmChartRelease(namespace, releaseName string) (*ReleaseWithWorkloadStatus, error) {
	actionConfig, err := SetupHelmConfiguration(namespace)
	if err != nil {
		return nil, err
	}

	get := action.NewGet(actionConfig)
	release, err := get.Run(releaseName)
	if err != nil {
		return nil, err
	}

	var manifestJSON []interface{}

	// Convert the release.Manifest YAML into a JSON object
	manifestYAML := strings.Split(release.Manifest, "---")
	for _, resourceYAML := range manifestYAML {
		if strings.TrimSpace(resourceYAML) == "" {
			continue
		}

		var resource interface{}
		err := yaml.Unmarshal([]byte(resourceYAML), &resource)
		if err != nil {
			zap.L().Error("Failed to unmarshal YAML", zap.Error(err))
			continue
		}

		resourceBytes, err := json.Marshal(resource)
		if err != nil {
			zap.L().Error("Failed to marshal JSON", zap.Error(err))
			continue
		}

		var resourceJSON interface{}
		if err := json.Unmarshal(resourceBytes, &resourceJSON); err != nil {
			zap.L().Error("Failed to unmarshal JSON", zap.Error(err))
			continue
		}

		manifestJSON = append(manifestJSON, resourceJSON)
	}

	// Construct ReleaseResponse object
	response := ReleaseWithWorkloadStatus{
		release,
		manifestJSON,
		[]WorkloadStatus{},
	}

	return &response, nil
}

// GetHelmChartReleaseWithWorkloads returns a Helm chart release with the status of its workloads.
func GetHelmChartReleaseWithWorkloads(namespace, releaseName string) (*ReleaseWithWorkloadStatus, error) {
	release, err := GetHelmChartRelease(namespace, releaseName)
	if err != nil {
		return nil, err
	}

	statuses, err := fetchHelmChartharttWorkloads(release.Manifest.([]interface{}))
	if err != nil {
		return nil, err
	}

	release.Workloads = statuses

	return release, nil
}

// ListHelmChartReleases lists all Helm charts in the given namespace.
// Returns a slice of release.Release pointers and any error encountered.
func ListHelmChartReleases(namespace string) ([]*release.Release, error) {
	actionConfig, err := SetupHelmConfiguration(namespace) // empty namespace for cluster-wide scope
	if err != nil {
		return nil, err // Corrected error handling
	}

	list := action.NewList(actionConfig)
	list.All = true

	var response []*release.Release

	releases, err := list.Run()
	if err != nil {
		return nil, err
	}

	for _, rel := range releases {
		// minimize the response
		rel.Chart.Templates = nil
		rel.Chart.Schema = nil
		rel.Chart.Files = nil
		rel.Manifest = ""
		response = append(response, rel)
	}

	return response, nil
}

// SetupHelmConfiguration sets up the Helm configuration with the given namespace.
// Returns an action.Configuration object and any error encountered.
func SetupHelmConfiguration(namespace string) (*action.Configuration, error) {
	kubeConfig, err := GetKubernetesConfig()
	if err != nil {
		return nil, err
	}

	cfgFlags := genericclioptions.NewConfigFlags(false)
	cfgFlags.APIServer = &kubeConfig.Host
	cfgFlags.BearerToken = &kubeConfig.BearerToken
	cfgFlags.CAFile = &kubeConfig.CAFile

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(cfgFlags, namespace, os.Getenv("HELM_DRIVER"), nil); err != nil {
		return nil, err
	}

	return actionConfig, nil
}

func fetchHelmChartharttWorkloads(manifestJSON []interface{}) ([]WorkloadStatus, error) {
	kubeConfig, err := GetKubernetesConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	allowedKinds := map[string]bool{
		"Deployment":  true,
		"StatefulSet": true,
		"DaemonSet":   true,
		"Job":         true,
		"CronJob":     true,
	}

	var wg sync.WaitGroup
	statusCh := make(chan WorkloadStatus, len(manifestJSON))
	errCh := make(chan error, len(manifestJSON))

	for _, resource := range manifestJSON {
		resourceMap, ok := resource.(map[string]interface{})
		if !ok {
			continue
		}

		kind, ok := resourceMap["kind"].(string)
		if !ok || !allowedKinds[kind] {
			continue
		}

		wg.Add(1)
		go func(kind string, resourceMap map[string]interface{}) {
			defer wg.Done()
			processWorkload(kind, resourceMap, clientset, statusCh, errCh)
		}(kind, resourceMap)
	}

	go func() {
		wg.Wait()
		close(statusCh)
		close(errCh)
	}()

	var statuses []WorkloadStatus
	for status := range statusCh {
		statuses = append(statuses, status)
	}

	// Check for errors
	var returnErr error
	for err := range errCh {
		returnErr = err // You might want to aggregate errors
	}

	return statuses, returnErr
}

func processWorkload(kind string, resourceMap map[string]interface{}, clientset *kubernetes.Clientset, statusCh chan<- WorkloadStatus, errCh chan<- error) {
	name, namespace, err := extractMetadata(resourceMap)
	if err != nil {
		errCh <- err
		return
	}

	var labelSelector string
	ctx := context.Background()
	switch kind {
	case "Deployment":
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			errCh <- err
			return
		}
		statusCh <- WorkloadStatus{Name: name, Kind: kind, Status: deployment.Status}
		labelSelector = metav1.FormatLabelSelector(deployment.Spec.Selector)

	case "StatefulSet":
		statefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			errCh <- err
			return
		}
		statusCh <- WorkloadStatus{Name: name, Kind: kind, Status: statefulSet.Status}
		labelSelector = metav1.FormatLabelSelector(statefulSet.Spec.Selector)

	case "DaemonSet":
		daemonSet, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			errCh <- err
			return
		}
		statusCh <- WorkloadStatus{Name: name, Kind: kind, Status: daemonSet.Status}
		labelSelector = metav1.FormatLabelSelector(daemonSet.Spec.Selector)

	case "Job":
		job, err := clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			errCh <- err
			return
		}
		statusCh <- WorkloadStatus{Name: name, Kind: kind, Status: job.Status}
		labelSelector = metav1.FormatLabelSelector(job.Spec.Selector)

	case "CronJob":
		cronJob, err := clientset.BatchV1beta1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			errCh <- err
			return
		}
		statusCh <- WorkloadStatus{Name: name, Kind: kind, Status: cronJob.Status}
		labelSelector = metav1.FormatLabelSelector(cronJob.Spec.JobTemplate.Spec.Selector)

	default:
		errCh <- fmt.Errorf("unsupported kind: %s", kind)
	}

	// Fetch and process pods if labelSelector is set
	if labelSelector != "" {
		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			errCh <- err
			return
		}
		for i := range pods.Items {
			statusCh <- WorkloadStatus{Name: pods.Items[i].Name, Kind: "Pod", Status: pods.Items[i].Status}
		}
	}

}

func extractMetadata(resourceMap map[string]interface{}) (string, string, error) {
	metadata, ok := resourceMap["metadata"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("metadata not found")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return "", "", fmt.Errorf("name not found in metadata")
	}

	// Get namespace from metadata, default to metav1.NamespaceAll if not found
	namespace, ok := metadata["namespace"].(string)
	if !ok {
		namespace = metav1.NamespaceAll
	}

	return name, namespace, nil
}
