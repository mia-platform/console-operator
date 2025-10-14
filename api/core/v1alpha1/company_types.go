/*
Copyright 2025.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type Proxy struct {
	Url string `json:"url"`
}

type ClusterConnection struct {
	Url                 string     `json:"url"`
	Base64CA            string     `json:"base64CA,omitempty"`
	Proxy               string     `json:"proxy,omitempty"`
	ServiceAccountToken Credential `json:"serviceAccountToken,omitempty"`
}

type Cluster struct {
	ClusterId   string            `json:"clusterId"`
	Connection  ClusterConnection `json:"clusterConnection"`
	Description string            `json:"description,omitempty"`
}

// CompanySpec defines the desired state of Company.
type CompanySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Company. Edit company_types.go to remove/update
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	CompanyOwners []string       `json:"companyOwners,omitempty"`
	ConsoleRef    NamespacedName `json:"consoleRef"`
	Clusters      []Cluster      `json:"clusters,omitempty"`
	// Clusters []string `json:"companyOwners"`
	// Envs []string `json:"companyOwners"`
}

// CompanyStatus defines the observed state of Company.
type CompanyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	CompanyId   string `json:"companyId,omitempty"`
	CompanyName string `json:"companyName,omitempty"`
	Exists      bool   `json:"exists,omitempty"`
	Description string `json:"description,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Exists",type=boolean,JSONPath=`.status.exists`
// +kubebuilder:printcolumn:name="Company Name",type=string,JSONPath=`.status.companyName`
// +kubebuilder:printcolumn:name="Company Id",type=string,JSONPath=`.status.companyId`

// Company is the Schema for the companies API.
type Company struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CompanySpec   `json:"spec,omitempty"`
	Status CompanyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CompanyList contains a list of Company.
type CompanyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Company `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Company{}, &CompanyList{})
}
