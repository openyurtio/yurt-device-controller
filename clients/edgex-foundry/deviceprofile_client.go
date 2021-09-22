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

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
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
