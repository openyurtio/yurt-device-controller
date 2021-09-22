/*
Copyright 2021 The OpenYurt Authors.

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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

const (
	DeviceFinalizer = "v1alpha1.device.finalizer"
	// DeviceSyncedCondition indicates that the device exists in both OpenYurt and edge platform
	DeviceSyncedCondition clusterv1.ConditionType = "DeviceSynced"
	// DeviceManagingCondition indicates that the device is being managed by cloud and its properties are being reconciled
	DeviceManagingCondition clusterv1.ConditionType = "DeviceManaging"
)

type AdminState string

const (
	Locked   AdminState = "LOCKED"
	UnLocked AdminState = "UNLOCKED"
)

type OperatingState string

const (
	Enabled  OperatingState = "ENABLED"
	Disabled OperatingState = "DISABLED"
)

type ProtocolProperties map[string]string

// DeviceSpec defines the desired state of Device
type DeviceSpec struct {
	// Information describing the device
	Description string `json:"description,omitempty"`
	// Admin state (locked/unlocked)
	AdminState AdminState `json:"adminState,omitempty"`
	// Operating state (enabled/disabled)
	OperatingState OperatingState `json:"operatingState,omitempty"`
	// A map of supported protocols for the given device
	Protocols map[string]ProtocolProperties `json:"protocols,omitempty"`
	// Other labels applied to the device to help with searching
	Labels []string `json:"labels,omitempty"`
	// Device service specific location (interface{} is an empty interface so
	// it can be anything)
	Location string `json:"location,omitempty"`
	// Associated Device Service - One per device
	Service string `json:"service"`
	// Associated Device Profile - Describes the device
	Profile string `json:"profile"`
	// True means device is managed by cloud, cloud can update the related fields
	// False means cloud can't update the fields
	Managed bool `json:"managed,omitempty"`
	// NodePool indicates which nodePool the device comes from
	NodePool string `json:"nodePool,omitempty"`
	// TODO support the following field
	// A list of auto-generated events coming from the device
	// AutoEvents     []AutoEvent                   `json:"autoEvents"`
	// DeviceProperties represents the expected state of the device's properties
	DeviceProperties map[string]DesiredPropertyState `json:"deviceProperties,omitempty"`
}

type DesiredPropertyState struct {
	Name         string `json:"name"`
	PutURL       string `json:"putURL,omitempty"`
	DesiredValue string `json:"desiredValue"`
}

type ActualPropertyState struct {
	Name        string `json:"name"`
	GetURL      string `json:"getURL,omitempty"`
	ActualValue string `json:"actualValue"`
}

// DeviceStatus defines the observed state of Device
type DeviceStatus struct {
	// Time (milliseconds) that the device last provided any feedback or
	// responded to any request
	LastConnected int64 `json:"lastConnected,omitempty"`
	// Time (milliseconds) that the device reported data to the core
	// microservice
	LastReported int64 `json:"lastReported,omitempty"`
	// Synced indicates whether the device already exists on both OpenYurt and edge platform
	Synced bool `json:"synced,omitempty"`
	// it represents the actual state of the device's properties
	DeviceProperties map[string]ActualPropertyState `json:"deviceProperties,omitempty"`
	EdgeId           string                         `json:"edgeId,omitempty"`
	// Admin state (locked/unlocked)
	AdminState AdminState `json:"adminState,omitempty"`
	// Operating state (enabled/disabled)
	OperatingState OperatingState `json:"operatingState,omitempty"`
	// current device state
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Device is the Schema for the devices API
type Device struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceSpec   `json:"spec,omitempty"`
	Status DeviceStatus `json:"status,omitempty"`
}

func (d *Device) SetConditions(conditions clusterv1.Conditions) {
	d.Status.Conditions = conditions
}

func (d *Device) GetConditions() clusterv1.Conditions {
	return d.Status.Conditions
}

//+kubebuilder:object:root=true

// DeviceList contains a list of Device
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Device `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Device{}, &DeviceList{})
}
