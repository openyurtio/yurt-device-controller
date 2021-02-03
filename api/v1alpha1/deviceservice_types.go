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

type Addressable struct {
	// ID is a unique identifier for the Addressable, such as a UUID
	Id string `json:"id,omitempty"`
	// Name is a unique name given to the Addressable
	Name string `json:"name,omitempty"`
	// Protocol for the address (HTTP/TCP)
	Protocol string `json:"protocol,omitempty"`
	// Method for connecting (i.e. POST)
	HTTPMethod string `json:"method,omitempty"`
	// Address of the addressable
	Address string `json:"address,omitempty"`
	// Port for the address
	Port int `json:"port,omitempty"`
	// Path for callbacks
	Path string `json:"path,omitempty"`
	// For message bus protocols
	Publisher string `json:"publisher,omitempty"`
	// User id for authentication
	User string `json:"user,omitempty"`
	// Password of the user for authentication for the addressable
	Password string `json:"password,omitempty"`
	// Topic for message bus addressables
	Topic string `json:"topic,omitempty"`
}

// DeviceServiceSpec defines the desired state of DeviceService
type DeviceServiceSpec struct {
	Description string `json:"description,omitempty"`
	// the Id assigned by the EdgeX foundry
	// TODO store this field in the status
	Id string `json:"id,omitempty"`
	// TODO store this field in the status
	LastConnected int64 `json:"lastConnected,omitempty"`
	// time in milliseconds that the device last reported data to the core
	// TODO store this field in the status
	LastReported int64 `json:"lastReported,omitempty"`
	// operational state - either enabled or disabled
	OperatingState OperatingState `json:"operatingState,omitempty"`
	// tags or other labels applied to the device service for search or other
	// identification needs on the EdgeX Foundry
	Labels []string `json:"labels,omitempty"`
	// address (MQTT topic, HTTP address, serial bus, etc.) for reaching
	// the service
	Addressable Addressable `json:"addressable,omitempty"`
	// Device Service Admin State
	AdminState AdminState `json:"adminState,omitempty"`
}

// DeviceServiceStatus defines the observed state of DeviceService
type DeviceServiceStatus struct {
	AddedToEdgeX bool `json:"addedToEdgeX,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DeviceService is the Schema for the deviceservices API
type DeviceService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceServiceSpec   `json:"spec,omitempty"`
	Status DeviceServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeviceServiceList contains a list of DeviceService
type DeviceServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeviceService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeviceService{}, &DeviceServiceList{})
}
