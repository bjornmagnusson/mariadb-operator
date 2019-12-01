/*

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Database defines the database
type Database struct {
	Name string `json:"name,omitempty"`
}

// DatabaseServerSpec defines the desired state of DatabaseServer
type DatabaseServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name      string     `json:"name,omitempty"`
	Image     string     `json:"image,omitempty"`
	Port      int32      `json:"port,omitempty"`
	Password  string     `json:"password,omitempty"`
	Databases []Database `json:"databases,omitempty"`
}

// DatabaseServerStatus defines the observed state of DatabaseServer
type DatabaseServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// DatabaseServer is the Schema for the databaseservers API
type DatabaseServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseServerSpec   `json:"spec,omitempty"`
	Status DatabaseServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseServerList contains a list of DatabaseServer
type DatabaseServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatabaseServer{}, &DatabaseServerList{})
}
