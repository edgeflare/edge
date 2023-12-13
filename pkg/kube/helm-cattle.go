package kube

import (
	"bytes"
	"context"
	"fmt"
	"io"

	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	"github.com/k3s-io/helm-controller/pkg/generated/clientset/versioned"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// ExtendedCattleHelmChart is a Helm chart with additional fields
type ExtendedCattleHelmChart struct {
	*helmv1.HelmChart
	InstallerJobLogs      string `json:"installer_job_logs,omitempty"`
	InstallerJobCompleted bool   `json:"installer_job_completed"`
}

// NewCattleHelmChartClient creates a new clientset for interacting with Cattle Helm Charts.
// It returns an error if the Kubernetes configuration cannot be created.
func NewCattleHelmChartClient() (*versioned.Clientset, error) {
	config, err := GetKubernetesConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// ListCattleHelmCharts lists Helm charts in the specified namespaces.
// If no namespace is provided, charts in all namespaces are listed.
func ListCattleHelmCharts(namespaces ...string) ([]helmv1.HelmChart, error) {
	clientset, err := NewCattleHelmChartClient()
	if err != nil {
		return nil, err
	}

	namespace := metav1.NamespaceAll
	if len(namespaces) > 0 {
		namespace = namespaces[0]
	}

	helmChartList, err := clientset.HelmV1().HelmCharts(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return helmChartList.Items, nil
}

// GetCattleHelmChart retrieves a specific Helm chart by name and namespace.
func GetCattleHelmChart(namespace string, name string) (*ExtendedCattleHelmChart, error) {
	helmClientset, err := NewCattleHelmChartClient()
	if err != nil {
		return nil, err
	}

	cattleHelmChart, err := helmClientset.HelmV1().HelmCharts(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	kubeConfig, err := GetKubernetesConfig()
	if err != nil {
		return nil, err
	}

	kubeClientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	job, err := kubeClientset.BatchV1().Jobs(namespace).Get(context.Background(), cattleHelmChart.Status.JobName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Check if job is complete and update cattleHelmChart
	extendedChart := ExtendedCattleHelmChart{
		HelmChart:             cattleHelmChart,
		InstallerJobCompleted: isCattleHelmChartInstallJobComplete(&job.Status),
	}

	// Add installer job logs if the job is not completed
	if !extendedChart.InstallerJobCompleted {
		logs, err := fetchInstallerJobLogs(namespace, job.Name)
		if err != nil {
			return nil, err
		}
		extendedChart.InstallerJobLogs = logs
	}

	return &extendedChart, nil
}

// CreateOrUpdateCattleHelmChart creates or updates a Helm chart.
// It automatically handles resource version conflicts.
func CreateOrUpdateCattleHelmChart(helmChart helmv1.HelmChart, namespaces ...string) (*helmv1.HelmChart, error) {
	clientset, err := NewCattleHelmChartClient()
	if err != nil {
		return nil, err
	}

	namespace := helmChart.Namespace
	if len(namespaces) > 0 {
		namespace = namespaces[0]
	}

	existing, err := clientset.HelmV1().HelmCharts(namespace).Get(context.Background(), helmChart.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// HelmChart doesn't exist, create it
		var created *helmv1.HelmChart
		created, err = clientset.HelmV1().HelmCharts(namespace).Create(context.Background(), &helmChart, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		return created, nil
	} else if err != nil {
		return nil, err
	}

	// HelmChart exists, update it
	helmChart.ResourceVersion = existing.ResourceVersion // Set the ResourceVersion to ensure a conflict-free update
	var updated *helmv1.HelmChart
	updated, err = clientset.HelmV1().HelmCharts(namespace).Update(context.Background(), &helmChart, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteCattleHelmChart deletes a Helm chart by name and namespace.
func DeleteCattleHelmChart(namespace string, name string) error {
	clientset, err := NewCattleHelmChartClient()
	if err != nil {
		return err
	}

	err = clientset.HelmV1().HelmCharts(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func isCattleHelmChartInstallJobComplete(jobStatus *v1.JobStatus) bool {
	if jobStatus == nil {
		return false
	}

	// Check if the job succeeded and did not fail
	if jobStatus.Succeeded > 0 && jobStatus.Failed == 0 {
		// Additionally, check the conditions for a 'Complete' status
		for _, condition := range jobStatus.Conditions {
			if condition.Type == "Complete" && condition.Status == "True" {
				return true
			}
		}
	}
	return false
}

func fetchInstallerJobLogs(namespace, jobName string) (string, error) {
	// Create a Kubernetes client
	config, err := GetKubernetesConfig()
	if err != nil {
		return "", err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	// Get pods in the namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set{"job-name": jobName}.String(), // Adjust label selector as needed
	})
	if err != nil {
		return "", err
	}

	// Find the specific pod (assuming the first pod is the installer pod)
	if len(pods.Items) == 0 {
		return "", err
	}
	podName := pods.Items[0].Name

	fmt.Println("podName", podName)

	// Get logs from the pod
	logOptions := &corev1.PodLogOptions{}
	podLogs, err := clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions).Stream(context.TODO())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	// Read logs from the stream
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	fmt.Println(buf.String())

	return buf.String(), nil
}
