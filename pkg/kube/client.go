package kube

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

// GetKubernetesConfig returns Kubernetes config
// If KUBECONFIG environment variable is set, it will use that file.
// If KUBECONFIG is not set, it will use ~/.kube/config file.
// Otherwise, it will use in-cluster configuration (ServiceAccount)
func GetKubernetesConfig() (*rest.Config, error) {
	// First, check if KUBECONFIG environment variable is set

	var err error
	var restConfig *rest.Config

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath != "" {
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
		return restConfig, nil
	}

	// If KUBECONFIG is not set, check the default kubeconfig path
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	kubeconfigPath = filepath.Join(home, ".kube", "config")

	if _, err = os.Stat(kubeconfigPath); err == nil {
		// If the default kubeconfig path exists, use it
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
		return restConfig, nil
	} else if os.IsNotExist(err) {
		// If the file does not exist, fall back to in-cluster config
		return rest.InClusterConfig()
	}

	return nil, err
}

// NewDynamicClient creates a new dynamic Kubernetes client.
func NewDynamicClient() (dynamic.Interface, error) {
	config, err := GetKubernetesConfig()
	if err != nil {
		return nil, err
	}

	return dynamic.NewForConfig(config)
}

// ApplyResource creates a Kubernetes resource from a YAML/JSON file or stdin.
// Or it updates the resource if the resource already exists.
func ApplyResource(fileContent []byte) error {
	// Decode YAML/JSON into unstructured.Unstructured
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(fileContent, nil, obj)
	if err != nil {
		return err
	}

	// Create the dynamic client
	dynamicClient, err := NewDynamicClient()
	if err != nil {
		return err
	}

	// Find the GroupVersionResource
	gvr, err := FindGVRFromGVK(*gvk)
	if err != nil {
		return err
	}

	// Prepare the resource for applying
	resourceClient := dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())

	// Apply the resource
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := resourceClient.Patch(context.TODO(), obj.GetName(), types.ApplyPatchType, fileContent, metav1.PatchOptions{
			FieldManager: "edge-manager",
		})
		return err
	})
}

// GetResources retrieves a list of Kubernetes resources based on the resource name and namespace.
// If namespace is an empty string, it lists resources across all namespaces.
func GetResources(resourceType, resourceName, namespace string) ([]unstructured.Unstructured, error) {
	dynamicClient, err := NewDynamicClient()
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	// Find the GroupVersionResource for the given resource type
	gvr, err := FindGVRFromResourceType(resourceType)
	if err != nil {
		return nil, fmt.Errorf("finding GVR from resource type '%s': %w", resourceType, err)
	}

	// If namespace is empty, list resources across all namespaces
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	var resources []unstructured.Unstructured
	if resourceName == "" {
		// List resources
		list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing resources for %v in namespace '%s': %w", gvr, namespace, err)
		}
		resources = list.Items
	} else {
		// Get specific resource
		resource, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("getting resource %s/%s: %w", resourceType, resourceName, err)
		}
		resources = append(resources, *resource)
	}

	return resources, nil
}

// DeleteResource deletes a Kubernetes resource.
// If resourceType and resourceName are provided, it deletes that specific resource.
// If fileContent is provided, it extracts the resource details from the file and then deletes it.
func DeleteResource(resourceType, resourceName, namespace string, fileContent []byte) error {
	var err error
	dynamicClient, err := NewDynamicClient()
	if err != nil {
		return fmt.Errorf("creating dynamic client: %w", err)
	}

	var gvr schema.GroupVersionResource
	if len(fileContent) > 0 {
		var gvk *schema.GroupVersionKind
		// Extract details from file
		decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		obj := &unstructured.Unstructured{}
		_, gvk, err = decUnstructured.Decode(fileContent, nil, obj)
		if err != nil {
			return fmt.Errorf("decoding resource: %w", err)
		}
		// resourceType = obj.GetKind()
		resourceName = obj.GetName()
		namespace = obj.GetNamespace()
		gvr, err = FindGVRFromGVK(*gvk)
		if err != nil {
			return err
		}
	} else {
		// Find the GVR from resource type
		gvr, err = FindGVRFromResourceType(resourceType)
		if err != nil {
			return fmt.Errorf("finding GVR: %w", err)
		}
	}

	return dynamicClient.Resource(gvr).Namespace(namespace).Delete(context.TODO(), resourceName, metav1.DeleteOptions{})
}

// FindGVRFromResourceName finds the GroupVersionResource for a given resource name using Kubernetes discovery.
func FindGVRFromResourceType(resourceType string) (schema.GroupVersionResource, error) {
	config, err := GetKubernetesConfig()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("getting kubernetes config: %w", err)
	}

	// Create a Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("creating kubernetes client: %w", err)
	}

	// Create a discovery client
	discoveryClient := clientset.Discovery()

	// Get available API resources
	apiResourceLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("getting server preferred resources: %w", err)
	}

	for _, apiResourceList := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}

		for i := range apiResourceList.APIResources {
			// Check the Name field, which is typically the plural form used in the API
			if strings.EqualFold(apiResourceList.APIResources[i].Name, resourceType) {
				return gv.WithResource(apiResourceList.APIResources[i].Name), nil
			}
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("resource %s not found", resourceType)
}

// FindGVRFromGVK finds the GroupVersionResource for a given GroupVersionKind using Kubernetes discovery.
func FindGVRFromGVK(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	config, err := GetKubernetesConfig()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("getting kubernetes config: %w", err)
	}

	// Create a Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("creating kubernetes client: %w", err)
	}

	// Create a discovery client
	discoveryClient := clientset.Discovery()

	// Get available API resources
	apiResourceLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("getting server preferred resources: %w", err)
	}

	for i := range apiResourceLists {
		for j := range apiResourceLists[i].APIResources {
			group, version := getGroupVersion(apiResourceLists[i].GroupVersion)
			if group == gvk.Group && version == gvk.Version && strings.EqualFold(apiResourceLists[i].APIResources[j].Kind, gvk.Kind) {
				return schema.GroupVersionResource{Group: group, Version: version, Resource: apiResourceLists[i].APIResources[j].Name}, nil
			}
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("GVR not found for GVK %s", gvk.String())
}

// getGroupVersion splits a groupVersion string into its group and version components.
func getGroupVersion(groupVersion string) (string, string) {
	parts := strings.Split(groupVersion, "/")
	if len(parts) == 1 {
		return "", parts[0] // Core group (""), version (e.g., "v1")
	}
	return parts[0], parts[1] // Group (e.g., "apps"), version (e.g., "v1")
}
