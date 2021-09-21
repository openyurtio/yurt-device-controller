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

package edgex_foundry

import (
	"strings"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EdgeXObjectName   = "device-controller/edgex-object.name"
	DeviceServicePath = "/api/v1/deviceservice"
	DeviceProfilePath = "/api/v1/deviceprofile"
	AddressablePath   = "/api/v1/addressable"
)

type ClientURL struct {
	Host string
	Port int
}

func getEdgeDeviceServiceName(ds *devicev1alpha1.DeviceService) string {
	var actualDSName string
	if _, ok := ds.ObjectMeta.Labels[EdgeXObjectName]; ok {
		actualDSName = ds.ObjectMeta.Labels[EdgeXObjectName]
	} else {
		actualDSName = ds.GetName()
	}
	return actualDSName
}

func toEdgexDeviceService(ds *devicev1alpha1.DeviceService) models.DeviceService {
	return models.DeviceService{
		DescribedObject: models.DescribedObject{
			Description: ds.Spec.Description,
		},
		Name: ds.GetName(),
		//Id:             ds.Spec.Id,
		LastConnected:  ds.Status.LastConnected,
		LastReported:   ds.Status.LastReported,
		OperatingState: models.OperatingState(ds.Spec.OperatingState),
		Labels:         ds.Spec.Labels,
		AdminState:     models.AdminState(ds.Spec.AdminState),
		Addressable:    toEdgeXAddressable(&ds.Spec.Addressable),
	}
}

func toEdgeXAddressable(a *devicev1alpha1.Addressable) models.Addressable {
	return models.Addressable{
		Id:         a.Id,
		Name:       a.Name,
		Protocol:   a.Protocol,
		HTTPMethod: a.HTTPMethod,
		Address:    a.Address,
		Port:       a.Port,
		Path:       a.Path,
		Publisher:  a.Publisher,
		User:       a.User,
		Password:   a.Password,
		Topic:      a.Topic,
	}
}

func toKubeDeviceService(ds models.DeviceService) devicev1alpha1.DeviceService {
	return devicev1alpha1.DeviceService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ds.Name),
			Namespace: "default",
			Labels: map[string]string{
				EdgeXObjectName: ds.Name,
			},
		},
		Spec: devicev1alpha1.DeviceServiceSpec{
			Description:    ds.Description,
			OperatingState: toKubeOperatingState(ds.OperatingState),
			Labels:         ds.Labels,
			Addressable:    toKubeAddressable(ds.Addressable),
			AdminState:     toKubeAdminState(ds.AdminState),
		},
		Status: devicev1alpha1.DeviceServiceStatus{
			EdgeId:        ds.Id,
			LastConnected: ds.LastConnected,
			LastReported:  ds.LastReported,
			AdminState:    toKubeAdminState(ds.AdminState),
		},
	}
}

func toKubeAddressable(ad models.Addressable) devicev1alpha1.Addressable {
	return devicev1alpha1.Addressable{
		Id:         ad.Id,
		Name:       ad.Name,
		Protocol:   ad.Protocol,
		HTTPMethod: ad.HTTPMethod,
		Address:    ad.Address,
		Port:       ad.Port,
		Path:       ad.Path,
		Publisher:  ad.Publisher,
		User:       ad.User,
		Password:   ad.Password,
		Topic:      ad.Topic,
	}
}

// toKubeDevice serialize the EdgeX AdminState to the corresponding Kubernetes AdminState
func toKubeAdminState(ea models.AdminState) devicev1alpha1.AdminState {
	if ea == models.Locked {
		return devicev1alpha1.Locked
	}
	return devicev1alpha1.UnLocked
}

// toKubeDevice serialize the EdgeX OperatingState to the corresponding
// Kubernetes OperatingState
func toKubeOperatingState(ea models.OperatingState) devicev1alpha1.OperatingState {
	if ea == models.Enabled {
		return devicev1alpha1.Enabled
	}
	return devicev1alpha1.Disabled
}
