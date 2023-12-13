package kube

import (
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubernetesConfig returns Kubernetes config
// If KUBECONFIG environment variable is set, it will use that file
// Otherwise, it will use in-cluster configuration (ServiceAccount)
func GetKubernetesConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath != "" {
		restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
		return restConfig, nil
	}
	return rest.InClusterConfig()
}
