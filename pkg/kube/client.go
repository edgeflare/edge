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
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client provides dynamic interactions with Kubernetes
type Client struct {
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
}

// NewClient creates a new instance of Client
func NewClient() (*Client, error) {
	config, err := GetKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	return &Client{
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
	}, nil
}

// GetKubernetesConfig returns Kubernetes configuration
// It tries to read from KUBECONFIG, fallback to default kubeconfig path, or use in-cluster config
func GetKubernetesConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	kubeconfigPath = filepath.Join(home, ".kube", "config")
	if _, err = os.Stat(kubeconfigPath); err == nil {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else if os.IsNotExist(err) {
		return rest.InClusterConfig()
	}

	return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
}

// ApplyResource applies a Kubernetes resource from a YAML/JSON content
func (kc *Client) ApplyResource(fileContent []byte) error {
	// Decode YAML/JSON into unstructured.Unstructured
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(fileContent, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode resource: %w", err)
	}

	gvr, err := kc.findGVRFromGVK(*gvk)
	if err != nil {
		return fmt.Errorf("failed to find GVR from GVK: %w", err)
	}

	resourceClient := kc.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	_, err = resourceClient.Patch(context.TODO(), obj.GetName(), types.ApplyPatchType, fileContent, metav1.PatchOptions{
		FieldManager: "kube-client",
	})
	return err
}

// GetResources retrieves a list of Kubernetes resources based on the resource type, name, and namespace.
// If namespace is an empty string, it lists resources across all namespaces.
func (kc *Client) GetResources(resourceType, resourceName, namespace string) ([]unstructured.Unstructured, error) {
	gvr, err := kc.findGVRFromResourceType(resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to find GVR from resource type '%s': %w", resourceType, err)
	}

	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	var resources []unstructured.Unstructured
	if resourceName == "" {
		// List resources across namespaces or in a specific namespace
		list, err := kc.dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list resources for %v in namespace '%s': %w", gvr, namespace, err)
		}
		resources = list.Items
	} else {
		// Get a specific resource
		resource, err := kc.dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get resource %s/%s in namespace '%s': %w", resourceType, resourceName, namespace, err)
		}
		resources = append(resources, *resource)
	}

	return resources, nil
}

// findGVRFromResourceType finds the GroupVersionResource for a given resource type using Kubernetes discovery.
func (kc *Client) findGVRFromResourceType(resourceType string) (schema.GroupVersionResource, error) {
	apiResourceLists, err := kc.discoveryClient.ServerPreferredResources()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("failed to list server preferred resources: %w", err)
	}

	for i := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceLists[i].GroupVersion)
		if err != nil {
			continue // Ignore parse errors and try other resources
		}

		for j := range apiResourceLists[i].APIResources {
			if strings.EqualFold(apiResourceLists[i].APIResources[j].Name, resourceType) {
				return schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: apiResourceLists[i].APIResources[j].Name}, nil
			}
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("resource type %s not found", resourceType)
}

// findGVRFromGVK finds the GroupVersionResource for a given GroupVersionKind
func (kc *Client) findGVRFromGVK(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	apiResourceLists, err := kc.discoveryClient.ServerPreferredResources()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("failed to list server preferred resources: %w", err)
	}

	for i := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceLists[i].GroupVersion)
		if err != nil {
			continue // Ignore parse errors and try other resources
		}

		for j := range apiResourceLists[i].APIResources {
			if gv.Group == gvk.Group && gv.Version == gvk.Version && strings.EqualFold(apiResourceLists[i].APIResources[j].Kind, gvk.Kind) {
				return schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: apiResourceLists[i].APIResources[j].Name}, nil
			}
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("GVR not found for GVK %s", gvk.String())
}

// DeleteResource deletes a Kubernetes resource.
// If resourceType and resourceName are provided, it deletes that specific resource.
// If fileContent is provided, it extracts the resource details from the file and then deletes it.
func (kc *Client) DeleteResource(resourceType, resourceName, namespace string, fileContent []byte) error {
	var gvr schema.GroupVersionResource
	var err error

	if len(fileContent) > 0 {
		// Extract details from fileContent
		gvr, resourceName, namespace, err = kc.extractDetailsFromFile(fileContent)
		if err != nil {
			return fmt.Errorf("failed to extract details from file: %w", err)
		}
	} else {
		// Find the GVR from resource type
		gvr, err = kc.findGVRFromResourceType(resourceType)
		if err != nil {
			return fmt.Errorf("failed to find GVR from resource type '%s': %w", resourceType, err)
		}
	}

	return kc.dynamicClient.Resource(gvr).Namespace(namespace).Delete(context.TODO(), resourceName, metav1.DeleteOptions{})
}

// extractDetailsFromFile extracts the GVR, resource name, and namespace from a given file content
func (kc *Client) extractDetailsFromFile(fileContent []byte) (schema.GroupVersionResource, string, string, error) {
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(fileContent, nil, obj)
	if err != nil {
		return schema.GroupVersionResource{}, "", "", fmt.Errorf("failed to decode resource from file: %w", err)
	}

	gvr, err := kc.findGVRFromGVK(*gvk)
	if err != nil {
		return schema.GroupVersionResource{}, "", "", fmt.Errorf("failed to find GVR from GVK: %w", err)
	}

	return gvr, obj.GetName(), obj.GetNamespace(), nil
}

// ListAPIResources lists all available API resources
func (kc *Client) ListAPIResources() ([]*metav1.APIResourceList, error) {
	apiResourceList, err := kc.discoveryClient.ServerPreferredResources()
	if err != nil {
		panic(err.Error())
	}

	// for i := range apiResourceList {
	// 	for j := range apiResourceList[i].APIResources {
	// 		fmt.Printf("Name: %v, Kind: %v\n", apiResourceList[i].APIResources[j].Name, apiResourceList[i].APIResources[j].Kind)
	// 	}
	// }

	return apiResourceList, nil
}
