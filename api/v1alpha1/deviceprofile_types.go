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

type DeviceResource struct {
	Description string            `json:"description"`
	Name        string            `json:"name"`
	Tag         string            `json:"tag,omitempty"`
	Properties  ProfileProperty   `json:"properties"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

type ProfileProperty struct {
	Value PropertyValue `json:"value"`
	Units Units         `json:"units,omitempty"`
}

type PropertyValue struct {
	Type         string `json:"type,omitempty"`         // ValueDescriptor Type of property after transformations
	ReadWrite    string `json:"readWrite,omitempty"`    // Read/Write Permissions set for this property
	Minimum      string `json:"minimum,omitempty"`      // Minimum value that can be get/set from this property
	Maximum      string `json:"maximum,omitempty"`      // Maximum value that can be get/set from this property
	DefaultValue string `json:"defaultValue,omitempty"` // Default value set to this property if no argument is passed
	Size         string `json:"size,omitempty"`         // Size of this property in its type  (i.e. bytes for numeric types, characters for string types)
	Mask         string `json:"mask,omitempty"`         // Mask to be applied prior to get/set of property
	Shift        string `json:"shift,omitempty"`        // Shift to be applied after masking, prior to get/set of property
	Scale        string `json:"scale,omitempty"`        // Multiplicative factor to be applied after shifting, prior to get/set of property
	Offset       string `json:"offset,omitempty"`       // Additive factor to be applied after multiplying, prior to get/set of property
	Base         string `json:"base,omitempty"`         // Base for property to be applied to, leave 0 for no power operation (i.e. base ^ property: 2 ^ 10)
	// Required value of the property, set for checking error state. Failing an
	// assertion condition wil  l mark the device with an error state
	Assertion     string `json:"assertion,omitempty"`
	Precision     string `json:"precision,omitempty"`
	FloatEncoding string `json:"floatEncoding,omitempty"` // FloatEncoding indicates the representation of floating value of reading.  It should be 'Base64'   or 'eNotation'
	MediaType     string `json:"mediaType,omitempty"`
}

type Units struct {
	Type         string `json:"type,omitempty"`
	ReadWrite    string `json:"readWrite,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

type Command struct {
	// EdgeXId is a unique identifier used by EdgeX Foundry, such as a UUID
	EdgeXId string `json:"id,omitempty"`
	// Command name (unique on the profile)
	Name string `json:"name,omitempty"`
	// Get Command
	Get Get `json:"get,omitempty"`
	// Put Command
	Put Put `json:"put,omitempty"`
}

type Put struct {
	Action         `json:",inline"`
	ParameterNames []string `json:"parameterNames,omitempty"`
}

type Get struct {
	Action `json:",omitempty"`
}

type Action struct {
	// Path used by service for action on a device or sensor
	Path string `json:"path,omitempty"`
	// Responses from get or put requests to service
	Responses []Response `json:"responses,omitempty"`
	// Url for requests from command service
	URL string `json:"url,omitempty"`
}

// Response for a Get or Put request to a service
type Response struct {
	Code           string   `json:"code,omitempty"`
	Description    string   `json:"description,omitempty"`
	ExpectedValues []string `json:"expectedValues,omitempty"`
}

type ProfileResource struct {
	Name string              `json:"name,omitempty"`
	Get  []ResourceOperation `json:"get,omitempty"`
	Set  []ResourceOperation `json:"set,omitempty"`
}

type ResourceOperation struct {
	Index     string `json:"index,omitempty"`
	Operation string `json:"operation,omitempty"`
	// Deprecated
	Object string `json:"object,omitempty"`
	// The replacement of Object field
	DeviceResource string `json:"deviceResource,omitempty"`
	Parameter      string `json:"parameter,omitempty"`
	// Deprecated
	Resource string `json:"resource,omitempty"`
	// The replacement of Resource field
	DeviceCommand string            `json:"deviceCommand,omitempty"`
	Secondary     []string          `json:"secondary,omitempty"`
	Mappings      map[string]string `json:"mappings,omitempty"`
}

// DeviceProfileSpec defines the desired state of DeviceProfile
type DeviceProfileSpec struct {
	Description string `json:"description,omitempty"`
	// Manufacturer of the device
	Manufacturer string `json:"manufacturer,omitempty"`
	// Model of the device
	Model string `json:"model,omitempty"`
	// EdgeXLabels used to search for groups of profiles on EdgeX Foundry
	EdgeXLabels     []string         `json:"labels,omitempty"`
	DeviceResources []DeviceResource `json:"deviceResources,omitempty"`

	// TODO support the following field
	DeviceCommands []ProfileResource `json:"deviceCommands,omitempty"`
	CoreCommands   []Command         `json:"coreCommands,omitempty"`
}

// DeviceProfileStatus defines the observed state of DeviceProfile
type DeviceProfileStatus struct {
	EdgeXId      string `json:"id,omitempty"`
	AddedToEdgeX bool   `json:"addedToEdgeX,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DeviceProfile represents the attributes and operational capabilities of a device.
// It is a template for which there can be multiple matching devices within a given system.
// NOTE This struct is derived from
// edgex/go-mod-core-contracts/models/deviceprofile.go
type DeviceProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceProfileSpec   `json:"spec,omitempty"`
	Status DeviceProfileStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeviceProfileList contains a list of DeviceProfile
type DeviceProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeviceProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeviceProfile{}, &DeviceProfileList{})
}
