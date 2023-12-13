package kube

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetNamespaces returns a list of kubernetes namespaces
func GetNamespaces() ([]string, error) {
	kubeConfig, err := GetKubernetesConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var namespacesSlice []string
	for i := range namespaces.Items {
		namespacesSlice = append(namespacesSlice, namespaces.Items[i].Name)
	}

	return namespacesSlice, nil
}
