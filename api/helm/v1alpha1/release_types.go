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
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReleaseSpec defines the desired state of Release.
type ReleaseSpec struct {
	// ChartURL is the OCI reference to the Helm chart
	// +kubebuilder:validation:Required
	// example: registry-1.docker.io/bitnamicharts/postgresql:16.4.1
	ChartURL string `json:"chartURL"`
	// ValuesContent is a string representation of the values.yaml file
	// +optional
	ValuesContent string `json:"valuesContent,omitempty"`
}

// ReleaseStatus defines the observed state of Release.
type ReleaseStatus struct {
	// Status is the current state of the release
	HelmStatus release.Status `json:"helmStatus,omitempty"`
	// FirstDeployed is when the release was first deployed.
	FirstDeployed string `json:"first_deployed,omitempty"`
	// LastDeployed is when the release was last deployed.
	LastDeployed string `json:"last_deployed,omitempty"`
	// Deleted tracks when this object was deleted.
	Deleted string `json:"deleted"`
	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Release is the Schema for the releases API.
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseSpec   `json:"spec,omitempty"`
	Status ReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReleaseList contains a list of Release.
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
