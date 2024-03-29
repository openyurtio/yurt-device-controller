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

package util

import (
	"testing"

	devicev1alpha1 "github.com/openyurtio/device-controller/apis/device.openyurt.io/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

func TestGetNodePool(t *testing.T) {
	cfg := &rest.Config{}
	res, err := GetNodePool(cfg)
	if res != "" {
		t.Errorf("expect nil on null config")
	}
	if err == nil {
		t.Errorf("null config must cause error")
	}
}

func TestGetEdgeDeviceServiceName(t *testing.T) {
	d := &devicev1alpha1.DeviceService{}
	assert.Equal(t, GetEdgeDeviceServiceName(d, ""), "")
	assert.Equal(t, GetEdgeDeviceServiceName(d, "a"), "")
}

func TestGetEdgeDeviceName(t *testing.T) {
	d := &devicev1alpha1.Device{}
	assert.Equal(t, GetEdgeDeviceName(d, ""), "")
	assert.Equal(t, GetEdgeDeviceName(d, "a"), "")
}

func TestGetEdgeDeviceProfileName(t *testing.T) {
	d := &devicev1alpha1.DeviceProfile{}
	assert.Equal(t, GetEdgeDeviceProfileName(d, ""), "")
	assert.Equal(t, GetEdgeDeviceProfileName(d, "a"), "")
}
