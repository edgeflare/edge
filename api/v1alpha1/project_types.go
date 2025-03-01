/*
Copyright 2025 edgeflare.io.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	helmv1alpha1 "github.com/edgeflare/edge/api/helm/v1alpha1"
	"github.com/edgeflare/edge/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// +optional
	Database *Database `json:"database,omitempty"`
	// +optional
	Auth *Auth `json:"auth,omitempty"`
	// +optional
	API *API `json:"api,omitempty"`
	// +optional
	Storage *Storage `json:"storage,omitempty"`
	// +optional
	PubSub *PubSub `json:"pubsub,omitempty"`
}

// ComponentRef defines a reference to an existing component or an external resource
type ComponentRef struct {
	// External references an external resource via secret
	// +optional
	External *ExternalRef `json:"external,omitempty"`
	// Release is Helm chart release. If release already exists, it's upgraded if old and new values differ
	// +optional
	Release *helmv1alpha1.ReleaseSpec `json:"release,omitempty"`
}

// GetSecretName returns the name of the secret for this component, if any
func (c *ComponentRef) GetSecretName() string {
	if c.External != nil {
		return c.External.SecretName
	}
	return ""
}

// IsExternal returns true if this component is referencing an external resource
func (c *ComponentRef) IsExternal() bool {
	return c.External != nil
}

// IsRelease returns true if this component should be deployed as a Helm release
func (c *ComponentRef) IsRelease() bool {
	return c.Release != nil || (c.External == nil && c.Release == nil)
}

// GetReleaseSpec returns the ReleaseSpec for this component
// If the release field is nil but the component should be a release,
// it returns a default ReleaseSpec.
func (c *ComponentRef) GetReleaseSpec(componentType, projectName string) helmv1alpha1.ReleaseSpec {
	if c.Release != nil {
		return *c.Release
	}

	// Default values based on component type
	chartURL := common.DefaultChartURL(componentType)
	valuesContent := common.DefaultValuesContent(componentType, projectName)

	return helmv1alpha1.ReleaseSpec{
		ChartURL:      chartURL,
		ValuesContent: valuesContent,
	}
}

// ExternalRef references external resources via Secret
type ExternalRef struct {
	// SecretName is the name of the secret containing credentials
	SecretName string `json:"secretName"`
}

// Database defines database configuration
type Database struct {
	// +optional
	Postgres *ComponentRef `json:"postgres,omitempty"`
}

// GetComponentRef returns the ComponentRef for the requested database type
func (d *Database) GetComponentRef(dbType string) *ComponentRef {
	switch dbType {
	case "postgres":
		if d.Postgres == nil {
			d.Postgres = &ComponentRef{}
		}
		return d.Postgres
	default:
		return nil
	}
}

// Auth defines identity provider configuration
type Auth struct {
	// +optional
	Zitadel *ComponentRef `json:"zitadel,omitempty"`
	// +optional
	Keycloak *ComponentRef `json:"keycloak,omitempty"`
}

// GetComponentRef returns the ComponentRef for the requested auth type
func (a *Auth) GetComponentRef(authType string) *ComponentRef {
	switch authType {
	case "zitadel":
		if a.Zitadel == nil {
			a.Zitadel = &ComponentRef{}
		}
		return a.Zitadel
	case "keycloak":
		if a.Keycloak == nil {
			a.Keycloak = &ComponentRef{}
		}
		return a.Keycloak
	default:
		return nil
	}
}

// API defines the API layer configuration
type API struct {
	// +optional
	PostgREST *ComponentRef `json:"postgrest,omitempty"`
}

// GetComponentRef returns the ComponentRef for the requested API type
func (a *API) GetComponentRef(apiType string) *ComponentRef {
	switch apiType {
	case "postgrest":
		if a.PostgREST == nil {
			a.PostgREST = &ComponentRef{}
		}
		return a.PostgREST
	default:
		return nil
	}
}

// Storage defines object storage configuration
type Storage struct {
	// +optional
	SeaweedFS *ComponentRef `json:"seaweedfs,omitempty"`
	// +optional
	Minio *ComponentRef `json:"minio,omitempty"`
}

// GetComponentRef returns the ComponentRef for the requested storage type
func (s *Storage) GetComponentRef(storageType string) *ComponentRef {
	switch storageType {
	case "seaweedfs":
		if s.SeaweedFS == nil {
			s.SeaweedFS = &ComponentRef{}
		}
		return s.SeaweedFS
	case "minio":
		if s.Minio == nil {
			s.Minio = &ComponentRef{}
		}
		return s.Minio
	default:
		return nil
	}
}

// PubSub defines pub/sub configuration
type PubSub struct {
	// +optional
	PGO *ComponentRef `json:"pgo,omitempty"`
}

// GetComponentRef returns the ComponentRef for the requested pubsub type
func (p *PubSub) GetComponentRef(pubsubType string) *ComponentRef {
	switch pubsubType {
	case "pgo":
		if p.PGO == nil {
			p.PGO = &ComponentRef{}
		}
		return p.PGO
	default:
		return nil
	}
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	// ObservedGeneration is the last generation that was reconciled
	Generation int64 `json:"generation,omitempty"`
	// Conditions represent the latest available observations of an object's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ComponentStatuses tracks the status of individual components
	// +optional
	ComponentStatuses map[string]ComponentStatus `json:"componentStatuses,omitempty"`
}

// ComponentStatus represents the status of an individual component
type ComponentStatus struct {
	// Ready indicates if the component is ready
	Ready bool `json:"ready"`
	// Message provides additional status information
	// +optional
	Message string `json:"message,omitempty"`
	// Endpoint where the component can be accessed
	// +optional
	Endpoint string `json:"endpoint,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// Project is the Schema for the projects API
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProjectSpec   `json:"spec,omitempty"`
	Status            ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}

// Helper functions for default values
