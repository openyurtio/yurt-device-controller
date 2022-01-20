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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
)

const (
	EdgeXObjectName     = "device-controller/edgex-object.name"
	DeviceServicePath   = "/api/v1/deviceservice"
	DeviceProfilePath   = "/api/v1/deviceprofile"
	AddressablePath     = "/api/v1/addressable"
	DevicePath          = "/api/v1/device"
	CommandResponsePath = "/api/v1/device"
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

func getEdgeDeviceName(d *devicev1alpha1.Device) string {
	var actualDeviceName string
	if _, ok := d.ObjectMeta.Labels[EdgeXObjectName]; ok {
		actualDeviceName = d.ObjectMeta.Labels[EdgeXObjectName]
	} else {
		actualDeviceName = d.GetName()
	}
	return actualDeviceName
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

func toEdgeXDeviceResourceSlice(drs []devicev1alpha1.DeviceResource) []models.DeviceResource {
	var ret []models.DeviceResource
	for _, dr := range drs {
		ret = append(ret, toEdgeXDeviceResource(dr))
	}
	return ret
}

func toEdgeXDeviceResource(dr devicev1alpha1.DeviceResource) models.DeviceResource {
	return models.DeviceResource{
		Description: dr.Description,
		Name:        dr.Name,
		Tag:         dr.Tag,
		Properties:  toEdgeXProfileProperty(dr.Properties),
		Attributes:  dr.Attributes,
	}
}

func toEdgeXProfileProperty(pp devicev1alpha1.ProfileProperty) models.ProfileProperty {
	return models.ProfileProperty{
		Value: toEdgeXPropertyValue(pp.Value),
		Units: toEdgeXUnits(pp.Units),
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

func toEdgeXPropertyValue(pv devicev1alpha1.PropertyValue) models.PropertyValue {
	return models.PropertyValue{
		Type:          pv.Type,
		ReadWrite:     pv.ReadWrite,
		Minimum:       pv.Minimum,
		Maximum:       pv.Maximum,
		DefaultValue:  pv.DefaultValue,
		Size:          pv.Size,
		Mask:          pv.Mask,
		Shift:         pv.Shift,
		Scale:         pv.Scale,
		Offset:        pv.Offset,
		Base:          pv.Base,
		Assertion:     pv.Assertion,
		Precision:     pv.Precision,
		FloatEncoding: pv.FloatEncoding,
		MediaType:     pv.MediaType,
	}
}

func toEdgeXUnits(u devicev1alpha1.Units) models.Units {
	return models.Units{
		Type:         u.Type,
		ReadWrite:    u.ReadWrite,
		DefaultValue: u.DefaultValue,
	}
}

func toEdgeXProfileResource(pr devicev1alpha1.ProfileResource) models.ProfileResource {
	gros := []models.ResourceOperation{}
	for _, gro := range pr.Get {
		gros = append(gros, toEdgeXResourceOperation(gro))
	}
	sros := []models.ResourceOperation{}
	for _, sro := range pr.Set {
		sros = append(sros, toEdgeXResourceOperation(sro))
	}
	return models.ProfileResource{
		Name: pr.Name,
		Get:  gros,
		Set:  sros,
	}
}

func toEdgeXResourceOperation(ro devicev1alpha1.ResourceOperation) models.ResourceOperation {
	return models.ResourceOperation{
		Index:          ro.Index,
		Operation:      ro.Operation,
		Object:         ro.Object,
		DeviceResource: ro.DeviceResource,
		Parameter:      ro.Parameter,
		Resource:       ro.Resource,
		DeviceCommand:  ro.DeviceCommand,
		Secondary:      ro.Secondary,
		Mappings:       ro.Mappings,
	}
}

func toEdgeXDevice(d *devicev1alpha1.Device) models.Device {
	md := models.Device{
		DescribedObject: models.DescribedObject{
			Description: d.Spec.Description,
		},
		Id:             d.Status.EdgeId,
		Name:           d.GetName(),
		AdminState:     toEdgeXAdminState(d.Spec.AdminState),
		OperatingState: toEdgeXOperatingState(d.Spec.OperatingState),
		Protocols:      toEdgeXProtocols(d.Spec.Protocols),
		LastConnected:  d.Status.LastConnected,
		LastReported:   d.Status.LastReported,
		Labels:         d.Spec.Labels,
		Location:       d.Spec.Location,
		Service:        models.DeviceService{Name: d.Spec.Service},
		Profile: toEdgeXDeviceProfile(
			devicev1alpha1.DeviceProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name: d.Spec.Profile,
				},
			},
		),
	}
	if d.Status.EdgeId != "" {
		md.Id = d.Status.EdgeId
	}
	return md
}

func toEdgeXProtocols(
	pps map[string]devicev1alpha1.ProtocolProperties) map[string]models.ProtocolProperties {
	ret := map[string]models.ProtocolProperties{}
	for k, v := range pps {
		ret[k] = models.ProtocolProperties(v)
	}
	return ret
}

func toEdgeXAdminState(as devicev1alpha1.AdminState) models.AdminState {
	if as == devicev1alpha1.Locked {
		return models.Locked
	}
	return models.Unlocked
}

func toEdgeXOperatingState(os devicev1alpha1.OperatingState) models.OperatingState {
	if os == devicev1alpha1.Enabled {
		return models.Enabled
	}
	return models.Disabled
}

func toEdgeXDeviceProfile(
	dp devicev1alpha1.DeviceProfile) models.DeviceProfile {
	cs := []models.Command{}
	for _, c := range dp.Spec.CoreCommands {
		cs = append(cs, toEdgeXCommand(c))
	}
	dcs := []models.ProfileResource{}
	for _, pr := range dp.Spec.DeviceCommands {
		dcs = append(dcs, toEdgeXProfileResource(pr))
	}

	return models.DeviceProfile{
		DescribedObject: models.DescribedObject{
			Description: dp.Spec.Description,
		},
		//Id:              dp.Spec.Id,
		Name:         dp.GetName(),
		Manufacturer: dp.Spec.Manufacturer,
		Model:        dp.Spec.Model,
		//Labels:          dp.Spec.Labels,
		DeviceResources: toEdgeXDeviceResourceSlice(dp.Spec.DeviceResources),
		DeviceCommands:  dcs,
		CoreCommands:    cs,
	}
}

func toEdgeXCommand(c devicev1alpha1.Command) models.Command {
	return models.Command{
		Id:   c.EdgeId,
		Name: c.Name,
		Get:  toEdgeXGet(c.Get),
		Put:  toEdgeXPut(c.Put),
	}
}
func toEdgeXPut(p devicev1alpha1.Put) models.Put {
	return models.Put{
		Action:         toEdgeXAction(p.Action),
		ParameterNames: p.ParameterNames,
	}
}

func toEdgeXGet(g devicev1alpha1.Get) models.Get {
	return models.Get{
		Action: toEdgeXAction(g.Action),
	}
}

func toEdgeXAction(a devicev1alpha1.Action) models.Action {
	responses := []models.Response{}
	for _, r := range a.Responses {
		responses = append(responses, toEdgeXResponse(r))
	}
	return models.Action{
		Path:      a.Path,
		Responses: responses,
		URL:       a.URL,
	}
}

func toEdgeXResponse(r devicev1alpha1.Response) models.Response {
	return models.Response{
		Code:           r.Code,
		Description:    r.Description,
		ExpectedValues: r.ExpectedValues,
	}
}

// toKubeDevice serialize the EdgeX Device to the corresponding Kubernetes Device
func toKubeDevice(ed models.Device) devicev1alpha1.Device {
	var loc string
	if ed.Location != nil {
		loc = ed.Location.(string)
	}
	return devicev1alpha1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ed.Name),
			Namespace: "default",
			Labels: map[string]string{
				EdgeXObjectName: ed.Name,
			},
		},
		Spec: devicev1alpha1.DeviceSpec{
			Description:    ed.Description,
			AdminState:     toKubeAdminState(ed.AdminState),
			OperatingState: toKubeOperatingState(ed.OperatingState),
			Protocols:      toKubeProtocols(ed.Protocols),
			Labels:         ed.Labels,
			Location:       loc,
			Service:        ed.Service.Name,
			Profile:        ed.Profile.Name,
		},
		Status: devicev1alpha1.DeviceStatus{
			LastConnected:  ed.LastConnected,
			LastReported:   ed.LastReported,
			Synced:         true,
			EdgeId:         ed.Id,
			AdminState:     toKubeAdminState(ed.AdminState),
			OperatingState: toKubeOperatingState(ed.OperatingState),
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

// toKubeProtocols serialize the EdgeX ProtocolProperties to the corresponding
// Kubernetes OperatingState
func toKubeProtocols(
	eps map[string]models.ProtocolProperties) map[string]devicev1alpha1.ProtocolProperties {
	ret := map[string]devicev1alpha1.ProtocolProperties{}
	for k, v := range eps {
		ret[k] = devicev1alpha1.ProtocolProperties(v)
	}
	return ret
}

// toKubeDeviceProfile create DeviceProfile in cloud according to devicProfile in edge
func toKubeDeviceProfile(dp *models.DeviceProfile) devicev1alpha1.DeviceProfile {
	return devicev1alpha1.DeviceProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(dp.Name),
			Namespace: "default",
			Labels: map[string]string{
				EdgeXObjectName: dp.Name,
			},
		},
		Spec: devicev1alpha1.DeviceProfileSpec{
			Description:     dp.Description,
			Manufacturer:    dp.Manufacturer,
			Model:           dp.Model,
			Labels:          dp.Labels,
			DeviceResources: toKubeDeviceResources(dp.DeviceResources),
			CoreCommands:    toKubeCoreCommands(dp.CoreCommands),
		},
		Status: devicev1alpha1.DeviceProfileStatus{
			EdgeId: dp.Id,
			Synced: true,
		},
	}
}

func toKubeDeviceResources(drs []models.DeviceResource) []devicev1alpha1.DeviceResource {
	var ret []devicev1alpha1.DeviceResource
	for _, dr := range drs {
		ret = append(ret, toKubeDeviceResource(dr))
	}
	return ret
}

func toKubeDeviceResource(dr models.DeviceResource) devicev1alpha1.DeviceResource {
	return devicev1alpha1.DeviceResource{
		Description: dr.Description,
		Name:        dr.Name,
		Tag:         dr.Tag,
		Properties:  toKubeProfileProperty(dr.Properties),
		Attributes:  dr.Attributes,
	}
}

func toKubeProfileProperty(pp models.ProfileProperty) devicev1alpha1.ProfileProperty {
	return devicev1alpha1.ProfileProperty{
		Value: toKubePropertyValue(pp.Value),
		Units: toKubeUnits(pp.Units),
	}
}

func toKubePropertyValue(pv models.PropertyValue) devicev1alpha1.PropertyValue {
	return devicev1alpha1.PropertyValue{
		Type:          pv.Type,
		ReadWrite:     pv.ReadWrite,
		Minimum:       pv.Minimum,
		Maximum:       pv.Maximum,
		DefaultValue:  pv.DefaultValue,
		Size:          pv.Size,
		Mask:          pv.Mask,
		Shift:         pv.Shift,
		Scale:         pv.Scale,
		Offset:        pv.Offset,
		Base:          pv.Base,
		Assertion:     pv.Assertion,
		Precision:     pv.Precision,
		FloatEncoding: pv.FloatEncoding,
		MediaType:     pv.MediaType,
	}
}

func toKubeUnits(u models.Units) devicev1alpha1.Units {
	return devicev1alpha1.Units{
		Type:         u.Type,
		ReadWrite:    u.ReadWrite,
		DefaultValue: u.DefaultValue,
	}
}

func toKubeCoreCommands(ccs []models.Command) []devicev1alpha1.Command {
	var ret []devicev1alpha1.Command
	for _, cc := range ccs {
		ret = append(ret, toKubeCoreCommand(cc))
	}
	return ret
}

func toKubeCoreCommand(cc models.Command) devicev1alpha1.Command {
	return devicev1alpha1.Command{
		Name:   cc.Name,
		EdgeId: cc.Id,
		Get:    toKubeGet(cc.Get),
		Put:    toKubePut(cc.Put),
	}
}

func toKubeGet(get models.Get) devicev1alpha1.Get {
	return devicev1alpha1.Get{
		Action: toKubeAction(get.Action),
	}
}

func toKubePut(put models.Put) devicev1alpha1.Put {
	return devicev1alpha1.Put{
		Action:         toKubeAction(put.Action),
		ParameterNames: put.ParameterNames,
	}
}

func toKubeAction(act models.Action) devicev1alpha1.Action {
	return devicev1alpha1.Action{
		Path:      act.Path,
		Responses: toKubeResponses(act.Responses),
		URL:       act.URL,
	}
}

func toKubeResponses(reps []models.Response) []devicev1alpha1.Response {
	var ret []devicev1alpha1.Response
	for _, rep := range reps {
		ret = append(ret, toKubeResponse(rep))
	}
	return ret
}

func toKubeResponse(rep models.Response) devicev1alpha1.Response {
	return devicev1alpha1.Response{
		Code:           rep.Code,
		Description:    rep.Description,
		ExpectedValues: rep.ExpectedValues,
	}
}

// ToEdgeXDeviceProfile create DeviceProfile in edge according to devicProfile in cloud
func ToEdgeXDeviceProfile(dp *devicev1alpha1.DeviceProfile) *models.DeviceProfile {
	cs := []models.Command{}
	for _, c := range dp.Spec.CoreCommands {
		cs = append(cs, toEdgeXCommand(c))
	}
	dcs := []models.ProfileResource{}
	for _, pr := range dp.Spec.DeviceCommands {
		dcs = append(dcs, toEdgeXProfileResource(pr))
	}

	return &models.DeviceProfile{
		DescribedObject: models.DescribedObject{
			Description: dp.Spec.Description,
		},
		Name:            dp.GetName(),
		Manufacturer:    dp.Spec.Manufacturer,
		Model:           dp.Spec.Model,
		Labels:          dp.Spec.Labels,
		DeviceResources: toEdgeXDeviceResourceSlice(dp.Spec.DeviceResources),
		DeviceCommands:  dcs,
		CoreCommands:    cs,
	}
}
