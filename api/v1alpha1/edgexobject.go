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

type EdgeXObject interface {
	IsAddedToEdgeX() bool
}

func (vd *ValueDescriptor) IsAddedToEdgeX() bool {
	return vd.Status.AddedToEdgeX
}

func (dp *DeviceProfile) IsAddedToEdgeX() bool {
	return dp.Status.Synced
}

func (d *Device) IsAddedToEdgeX() bool {
	return d.Status.AddedToEdgeX
}