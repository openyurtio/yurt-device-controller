/*
Copyright 2021 The Kubernetes authors.

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

// ValueDescriptorSpec defines the desired state of ValueDescriptor
type ValueDescriptorSpec struct {
	Id            string   `json:"id,omitempty"`
	Created       int64    `json:"created,omitempty"`
	Description   string   `json:"description,omitempty"`
	Modified      int64    `json:"modified,omitempty"`
	Origin        int64    `json:"origin,omitempty"`
	Min           string   `json:"min,omitempty"`
	Max           string   `json:"max,omitempty"`
	DefaultValue  string   `json:"defaultValue,omitempty"`
	Type          string   `json:"type,omitempty"`
	UomLabel      string   `json:"uomLabel,omitempty"`
	Formatting    string   `json:"formatting,omitempty"`
	Labels        []string `json:"labels,omitempty"`
	MediaType     string   `json:"mediaType,omitempty"`
	FloatEncoding string   `json:"floatEncoding,omitempty"`
}

// ValueDescriptorStatus defines the observed state of ValueDescriptor
type ValueDescriptorStatus struct {
	// AddedToEdgeX indicates whether the object has been successfully
	// created on EdgeX Foundry
	AddedToEdgeX bool `json:"addedToEdgeX,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ValueDescriptor is the Schema for the valuedescriptors API
// NOTE Thie struct is derived from
// edgex/go-mod-core-contracts/models/value-descriptor.go
type ValueDescriptor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ValueDescriptorSpec   `json:"spec,omitempty"`
	Status ValueDescriptorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ValueDescriptorList contains a list of ValueDescriptor
type ValueDescriptorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ValueDescriptor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ValueDescriptor{}, &ValueDescriptorList{})
}
