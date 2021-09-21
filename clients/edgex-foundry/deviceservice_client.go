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
	"github.com/openyurtio/device-controller/api/v1alpha1"
	edgeCli "github.com/openyurtio/device-controller/clients"
)

type EdgexDeviceServiceClient struct {
	*resty.Client
	CoreMetaClient ClientURL
	logr.Logger
}

func NewEdgexDeviceServiceClient(coreMetaClient ClientURL, log logr.Logger) *EdgexDeviceServiceClient {
	return &EdgexDeviceServiceClient{
		Client:         resty.New(),
		CoreMetaClient: coreMetaClient,
		Logger:         log,
	}
}

// Create function sends a POST request to EdgeX to add a new deviceService
func (eds *EdgexDeviceServiceClient) Create(ctx context.Context, deviceservice *v1alpha1.DeviceService, options edgeCli.CreateOptions) (*v1alpha1.DeviceService, error) {
	ds := toEdgexDeviceService(deviceservice)
	eds.V(5).Info("will add the DeviceServices",
		"DeviceService", ds.Name)
	dpJson, err := json.Marshal(&ds)
	if err != nil {
		return nil, err
	}
	postPath := fmt.Sprintf("http://%s:%d%s",
		eds.CoreMetaClient.Host, eds.CoreMetaClient.Port, DeviceServicePath)
	resp, err := eds.R().
		SetBody(dpJson).Post(postPath)
	if err != nil {
		return nil, err
	} else if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("create deviceService on edgex foundry failed, the response is : %s", resp.Body())
	}
	createdDs := deviceservice.DeepCopy()
	createdDs.Status.EdgeId = string(resp.Body())
	return createdDs, err
}

// Delete function sends a request to EdgeX to delete a deviceService
func (eds *EdgexDeviceServiceClient) Delete(ctx context.Context, name string, option edgeCli.DeleteOptions) error {
	eds.V(5).Info("will delete the DeviceService",
		"DeviceService", name)
	delURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		eds.CoreMetaClient.Host, eds.CoreMetaClient.Port, DeviceServicePath, name)
	resp, err := eds.R().Delete(delURL)
	if err != nil {
		return err
	}
	if string(resp.Body()) != "true" {
		return errors.New(string(resp.Body()))
	}
	return nil
}

// Update is used to set the admin or operating state of the deviceService by unique name of the deviceService.
// TODO support to update other fields
func (eds *EdgexDeviceServiceClient) Update(ctx context.Context, ds *v1alpha1.DeviceService, options edgeCli.UpdateOptions) (*v1alpha1.DeviceService, error) {
	actualDSName := getEdgeDeviceServiceName(ds)
	putBaseURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		eds.CoreMetaClient.Host, eds.CoreMetaClient.Port, DeviceServicePath, actualDSName)
	if ds == nil {
		return nil, nil
	}
	if ds.Spec.AdminState != "" {
		amURL := fmt.Sprintf("%s/adminstate/%s", putBaseURL, ds.Spec.AdminState)
		if rep, err := resty.New().R().SetHeader("Content-Type", "application/json").Put(amURL); err != nil {
			return nil, err
		} else if rep.StatusCode() != http.StatusOK {
			return nil, fmt.Errorf("failed to update deviceService: %s, get response: %s", actualDSName, string(rep.Body()))
		}
	}
	if ds.Spec.OperatingState != "" {
		opURL := fmt.Sprintf("%s/opstate/%s", putBaseURL, ds.Spec.OperatingState)
		if rep, err := resty.New().R().
			SetHeader("Content-Type", "application/json").Put(opURL); err != nil {
			return nil, err
		} else if rep.StatusCode() != http.StatusOK {
			return nil, fmt.Errorf("failed to update deviceService: %s, get response: %s", actualDSName, string(rep.Body()))
		}
	}

	return ds, nil
}

// Get is used to query the deviceService information corresponding to the deviceService name
func (eds *EdgexDeviceServiceClient) Get(ctx context.Context, name string, options edgeCli.GetOptions) (*v1alpha1.DeviceService, error) {
	eds.V(5).Info("will get DeviceServices",
		"DeviceService", name)
	var ds v1alpha1.DeviceService
	getURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		eds.CoreMetaClient.Host, eds.CoreMetaClient.Port, DeviceServicePath, name)
	resp, err := eds.R().Get(getURL)
	if err != nil {
		return &ds, err
	}
	if string(resp.Body()) == "Item not found\n" ||
		strings.HasPrefix(string(resp.Body()), "no item found") {
		return &ds, errors.New("Item not found")
	}
	var dp models.DeviceService
	err = json.Unmarshal(resp.Body(), &dp)
	ds = toKubeDeviceService(dp)
	return &ds, err
}

// List is used to get all deviceService objects on edge platform
// The Hanoi version currently supports only a single label and does not support other filters
func (eds *EdgexDeviceServiceClient) List(ctx context.Context, options edgeCli.ListOptions) ([]v1alpha1.DeviceService, error) {
	eds.V(5).Info("will list DeviceServices")
	lp := fmt.Sprintf("http://%s:%d%s",
		eds.CoreMetaClient.Host, eds.CoreMetaClient.Port, DeviceServicePath)
	if options.LabelSelector != nil {
		if _, ok := options.LabelSelector["label"]; ok {
			lp = strings.Join([]string{lp, strings.Join([]string{"label", options.LabelSelector["label"]}, "/")}, "/")
		}
	}
	resp, err := eds.R().
		EnableTrace().
		Get(lp)
	if err != nil {
		return nil, err
	}
	dss := []models.DeviceService{}
	if err := json.Unmarshal(resp.Body(), &dss); err != nil {
		return nil, err
	}
	var res []v1alpha1.DeviceService
	for _, ds := range dss {
		res = append(res, toKubeDeviceService(ds))
	}
	return res, nil
}

// CreateAddressable function sends a POST request to EdgeX to add a new addressable
func (eds *EdgexDeviceServiceClient) CreateAddressable(ctx context.Context, addressable *v1alpha1.Addressable, options edgeCli.CreateOptions) (*v1alpha1.Addressable, error) {
	as := toEdgeXAddressable(addressable)
	eds.V(5).Info("will add the Addressables",
		"Addressable", as.Name)
	dpJson, err := json.Marshal(&as)
	if err != nil {
		return nil, err
	}
	postPath := fmt.Sprintf("http://%s:%d%s",
		eds.CoreMetaClient.Host, eds.CoreMetaClient.Port, AddressablePath)
	resp, err := eds.R().
		SetBody(dpJson).Post(postPath)
	if err != nil {
		return nil, err
	}
	createdAddr := addressable.DeepCopy()
	createdAddr.Id = string(resp.Body())
	return createdAddr, err
}

// DeleteAddressable function sends a request to EdgeX to delete a addressable
func (eds *EdgexDeviceServiceClient) DeleteAddressable(ctx context.Context, name string, options edgeCli.DeleteOptions) error {
	eds.V(5).Info("will delete the Addressable",
		"Addressable", name)
	delURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		eds.CoreMetaClient.Host, eds.CoreMetaClient.Port, AddressablePath, name)
	resp, err := eds.R().Delete(delURL)
	if err != nil {
		return err
	}
	if string(resp.Body()) != "true" {
		return errors.New(string(resp.Body()))
	}
	return nil
}

// UpdateAddressable is used to update the addressable on edgex foundry
func (eds *EdgexDeviceServiceClient) UpdateAddressable(ctx context.Context, device *v1alpha1.Addressable, options edgeCli.UpdateOptions) (*v1alpha1.Addressable, error) {
	return nil, nil
}

// GetAddressable is used to query the addressable information corresponding to the addressable name
func (eds *EdgexDeviceServiceClient) GetAddressable(ctx context.Context, name string, options edgeCli.GetOptions) (*v1alpha1.Addressable, error) {
	eds.V(5).Info("will get Addressables",
		"Addressable", name)
	var addressable v1alpha1.Addressable
	getURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		eds.CoreMetaClient.Host, eds.CoreMetaClient.Port, AddressablePath, name)
	resp, err := eds.R().Get(getURL)
	if err != nil {
		return &addressable, err
	}
	if string(resp.Body()) == "Item not found\n" {
		return &addressable, errors.New("Item not found")
	}
	var maddr models.Addressable
	err = json.Unmarshal(resp.Body(), &maddr)
	addressable = toKubeAddressable(maddr)
	return &addressable, err
}

// ListAddressables is used to get all addressable objects on edge platform
func (eds *EdgexDeviceServiceClient) ListAddressables(ctx context.Context, options edgeCli.ListOptions) ([]v1alpha1.Addressable, error) {
	eds.V(5).Info("will list Addressables")
	lp := fmt.Sprintf("http://%s:%d%s",
		eds.CoreMetaClient.Host, eds.CoreMetaClient.Port, AddressablePath)
	resp, err := eds.R().
		EnableTrace().
		Get(lp)
	if err != nil {
		return nil, err
	}
	ass := []models.Addressable{}
	if err := json.Unmarshal(resp.Body(), &ass); err != nil {
		return nil, err
	}
	var res []v1alpha1.Addressable
	for i := range ass {
		res = append(res, toKubeAddressable(ass[i]))
	}
	return res, nil
}
