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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openyurtio/device-controller/api/v1alpha1"
	devcli "github.com/openyurtio/device-controller/clients"
	strutil "github.com/openyurtio/device-controller/controllers/util/strings"
)

type EdgexDeviceProfile struct {
	*resty.Client
	Host string
	Port int
	logr.Logger
}

const (
	DeviceProfilePath = "/api/v1/deviceprofile"
	EdgeXObjectName   = "device-controller/edgex-object.name"
)

func NewEdgexDeviceProfile(host string, port int, log logr.Logger) *EdgexDeviceProfile {
	return &EdgexDeviceProfile{
		Client: resty.New(),
		Host:   host,
		Port:   port,
		Logger: log,
	}
}

func getListDeviceProfileURL(host string, port int, opts devcli.ListOptions) (string, error) {
	url := fmt.Sprintf("http://%s:%d%s", host, port, DeviceProfilePath)
	if len(opts.LabelSelector) > 1 {
		return url, fmt.Errorf("Multiple labels: list only support one label")
	}
	if len(opts.LabelSelector) > 0 && len(opts.LabelSelector) > 0 {
		return url, fmt.Errorf("Multi list options: list action can't use 'label' with 'manufacturer' or 'model'")
	}
	for _, v := range opts.LabelSelector {
		url = fmt.Sprintf("%s/label/%s", url, v)
	}

	listParameters := []string{"manufacturer", "model"}
	for k, v := range opts.FieldSelector {
		if !strutil.IsInStringLst(listParameters, k) {
			return url, fmt.Errorf("Invaild list options: %s", k)
		}
		url = fmt.Sprintf("%s/%s/%s", url, k, v)
	}
	return url, nil
}

func (cdc *EdgexDeviceProfile) List(ctx context.Context, opts devcli.ListOptions) ([]v1alpha1.DeviceProfile, error) {
	cdc.V(5).Info("will list DeviceProfiles")
	lp, err := getListDeviceProfileURL(cdc.Host, cdc.Port, opts)
	if err != nil {
		return nil, err
	}
	resp, err := cdc.R().EnableTrace().Get(lp)
	if err != nil {
		return nil, err
	}
	dps := []models.DeviceProfile{}
	if err := json.Unmarshal(resp.Body(), &dps); err != nil {
		return nil, err
	}
	deviceProfiles := make([]v1alpha1.DeviceProfile, len(dps))
	for i, dp := range dps {
		deviceProfiles[i] = toKubeDeviceProfile(&dp)
	}
	return deviceProfiles, nil
}

func (cdc *EdgexDeviceProfile) Get(ctx context.Context, name string, opts devcli.GetOptions) (*v1alpha1.DeviceProfile, error) {
	cdc.V(5).Info("will get DeviceProfiles", "DeviceProfile", name)
	var dp models.DeviceProfile
	getURL := fmt.Sprintf("http://%s:%d%s/name/%s", cdc.Host, cdc.Port, DeviceProfilePath, name)
	resp, err := cdc.R().Get(getURL)
	if err != nil {
		return nil, err
	}
	if string(resp.Body()) == "Item not found\n" {
		return nil, errors.New("Item not found")
	}
	if err = json.Unmarshal(resp.Body(), &dp); err != nil {
		return nil, err
	}
	kubedp := toKubeDeviceProfile(&dp)
	return &kubedp, nil
}

func (cdc *EdgexDeviceProfile) Create(ctx context.Context, deviceProfile *v1alpha1.DeviceProfile, opts devcli.CreateOptions) (*v1alpha1.DeviceProfile, error) {
	edgeDp := ToEdgeXDeviceProfile(deviceProfile)
	dpJson, err := json.Marshal(edgeDp)
	if err != nil {
		return nil, err
	}
	postURL := fmt.Sprintf("http://%s:%d%s", cdc.Host, cdc.Port, DeviceProfilePath)
	resp, err := cdc.R().SetBody(dpJson).Post(postURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("create edgex deviceProfile err: %s", string(resp.Body())) // 假定 resp.Body() 存了 msg 信息
	}
	deviceProfile.Status.EdgeId = string(resp.Body())
	deviceProfile.Status.Synced = true
	return deviceProfile, err
}

// TODO
func (cdc *EdgexDeviceProfile) Update(ctx context.Context, deviceProfile *v1alpha1.DeviceProfile, opts devcli.UpdateOptions) (*v1alpha1.DeviceProfile, error) {
	return nil, nil
}

func (cdc *EdgexDeviceProfile) Delete(ctx context.Context, name string, opts devcli.DeleteOptions) error {
	cdc.V(5).Info("will delete the DeviceProfile", "DeviceProfile", name)
	delURL := fmt.Sprintf("http://%s:%d%s/name/%s", cdc.Host, cdc.Port, DeviceProfilePath, name)
	resp, err := cdc.R().Delete(delURL)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("delete edgex deviceProfile err: %s", string(resp.Body())) // 假定 resp.Body() 存了 msg 信息
	}
	return nil
}

// toKubeDeviceProfile create DeviceProfile in cloud according to devicProfile in edge
func toKubeDeviceProfile(dp *models.DeviceProfile) v1alpha1.DeviceProfile {
	return v1alpha1.DeviceProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(dp.Name),
			Namespace: "default",
			Labels: map[string]string{
				EdgeXObjectName: dp.Name,
			},
		},
		Spec: v1alpha1.DeviceProfileSpec{
			Description:     dp.Description,
			Manufacturer:    dp.Manufacturer,
			Model:           dp.Model,
			Labels:          dp.Labels,
			DeviceResources: toKubeDeviceResources(dp.DeviceResources),
			CoreCommands:    toKubeCoreCommands(dp.CoreCommands),
		},
		Status: v1alpha1.DeviceProfileStatus{
			EdgeId: dp.Id,
			Synced: true,
		},
	}
}

func toKubeDeviceResources(drs []models.DeviceResource) []v1alpha1.DeviceResource {
	var ret []v1alpha1.DeviceResource
	for _, dr := range drs {
		ret = append(ret, toKubeDeviceResource(dr))
	}
	return ret
}

func toKubeDeviceResource(dr models.DeviceResource) v1alpha1.DeviceResource {
	return v1alpha1.DeviceResource{
		Description: dr.Description,
		Name:        dr.Name,
		Tag:         dr.Tag,
		Properties:  toKubeProfileProperty(dr.Properties),
		Attributes:  dr.Attributes,
	}
}

func toKubeProfileProperty(pp models.ProfileProperty) v1alpha1.ProfileProperty {
	return v1alpha1.ProfileProperty{
		Value: toKubePropertyValue(pp.Value),
		Units: toKubeUnits(pp.Units),
	}
}

func toKubePropertyValue(pv models.PropertyValue) v1alpha1.PropertyValue {
	return v1alpha1.PropertyValue{
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

func toKubeUnits(u models.Units) v1alpha1.Units {
	return v1alpha1.Units{
		Type:         u.Type,
		ReadWrite:    u.ReadWrite,
		DefaultValue: u.DefaultValue,
	}
}

func toKubeCoreCommands(ccs []models.Command) []v1alpha1.Command {
	var ret []v1alpha1.Command
	for _, cc := range ccs {
		ret = append(ret, toKubeCoreCommand(cc))
	}
	return ret
}

func toKubeCoreCommand(cc models.Command) v1alpha1.Command {
	return v1alpha1.Command{
		Name:   cc.Name,
		EdgeId: cc.Id,
		Get:    toKubeGet(cc.Get),
		Put:    toKubePut(cc.Put),
	}
}

func toKubeGet(get models.Get) v1alpha1.Get {
	return v1alpha1.Get{
		Action: toKubeAction(get.Action),
	}
}

func toKubePut(put models.Put) v1alpha1.Put {
	return v1alpha1.Put{
		Action:         toKubeAction(put.Action),
		ParameterNames: put.ParameterNames,
	}
}

func toKubeAction(act models.Action) v1alpha1.Action {
	return v1alpha1.Action{
		Path:      act.Path,
		Responses: toKubeResponses(act.Responses),
		URL:       act.URL,
	}
}

func toKubeResponses(reps []models.Response) []v1alpha1.Response {
	var ret []v1alpha1.Response
	for _, rep := range reps {
		ret = append(ret, toKubeResponse(rep))
	}
	return ret
}

func toKubeResponse(rep models.Response) v1alpha1.Response {
	return v1alpha1.Response{
		Code:           rep.Code,
		Description:    rep.Description,
		ExpectedValues: rep.ExpectedValues,
	}
}

// ToEdgeXDeviceProfile create DeviceProfile in edge according to devicProfile in cloud
func ToEdgeXDeviceProfile(dp *v1alpha1.DeviceProfile) *models.DeviceProfile {
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

func toEdgeXProfileResource(pr v1alpha1.ProfileResource) models.ProfileResource {
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

func toEdgeXResourceOperation(ro v1alpha1.ResourceOperation) models.ResourceOperation {
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

func toEdgeXDeviceResourceSlice(drs []v1alpha1.DeviceResource) []models.DeviceResource {
	var ret []models.DeviceResource
	for _, dr := range drs {
		ret = append(ret, toEdgeXDeviceResource(dr))
	}
	return ret
}

func toEdgeXDeviceResource(dr v1alpha1.DeviceResource) models.DeviceResource {
	return models.DeviceResource{
		Description: dr.Description,
		Name:        dr.Name,
		Tag:         dr.Tag,
		Properties:  toEdgeXProfileProperty(dr.Properties),
		Attributes:  dr.Attributes,
	}
}

func toEdgeXProfileProperty(pp v1alpha1.ProfileProperty) models.ProfileProperty {
	return models.ProfileProperty{
		Value: toEdgeXPropertyValue(pp.Value),
		Units: toEdgeXUnits(pp.Units),
	}
}

func toEdgeXPropertyValue(pv v1alpha1.PropertyValue) models.PropertyValue {
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

func toEdgeXUnits(u v1alpha1.Units) models.Units {
	return models.Units{
		Type:         u.Type,
		ReadWrite:    u.ReadWrite,
		DefaultValue: u.DefaultValue,
	}
}

func toEdgeXCommand(c v1alpha1.Command) models.Command {
	return models.Command{
		Id:   c.EdgeId,
		Name: c.Name,
		Get:  toEdgeXGet(c.Get),
		Put:  toEdgeXPut(c.Put),
	}
}
func toEdgeXPut(p v1alpha1.Put) models.Put {
	return models.Put{
		Action:         toEdgeXAction(p.Action),
		ParameterNames: p.ParameterNames,
	}
}

func toEdgeXGet(g v1alpha1.Get) models.Get {
	return models.Get{
		Action: toEdgeXAction(g.Action),
	}
}

func toEdgeXAction(a v1alpha1.Action) models.Action {
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

func toEdgeXResponse(r v1alpha1.Response) models.Response {
	return models.Response{
		Code:           r.Code,
		Description:    r.Description,
		ExpectedValues: r.ExpectedValues,
	}
}
