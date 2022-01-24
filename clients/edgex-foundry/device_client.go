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
	"github.com/go-resty/resty/v2"
	"k8s.io/klog/v2"

	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
	edgeCli "github.com/openyurtio/device-controller/clients"
)

type EdgexDeviceClient struct {
	*resty.Client
	CoreMetaAddr    string
	CoreCommandAddr string
}

func NewEdgexDeviceClient(coreMetaAddr, coreCommandAddr string) *EdgexDeviceClient {
	return &EdgexDeviceClient{
		Client:          resty.New(),
		CoreMetaAddr:    coreMetaAddr,
		CoreCommandAddr: coreCommandAddr,
	}
}

// Create function sends a POST request to EdgeX to add a new device
func (efc *EdgexDeviceClient) Create(ctx context.Context, device *devicev1alpha1.Device, options edgeCli.CreateOptions) (*devicev1alpha1.Device, error) {
	dp := toEdgeXDevice(device)
	klog.V(5).Infof("will add the Devices: %s", dp.Name)
	dpJson, err := json.Marshal(&dp)
	if err != nil {
		return nil, err
	}
	postPath := fmt.Sprintf("http://%s%s", efc.CoreMetaAddr, DevicePath)
	resp, err := efc.R().
		SetBody(dpJson).Post(postPath)
	if err != nil {
		return nil, err
	} else if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("create device on edgex foundry failed, the response is : %s", resp.Body())
	}

	createdDevice := device.DeepCopy()
	createdDevice.Status.EdgeId = string(resp.Body())
	return createdDevice, err
}

// Delete function sends a request to EdgeX to delete a device
func (efc *EdgexDeviceClient) Delete(ctx context.Context, name string, options edgeCli.DeleteOptions) error {
	klog.V(5).Infof("will delete the Device: %s", name)
	delURL := fmt.Sprintf("http://%s%s/name/%s", efc.CoreMetaAddr, DevicePath, name)
	resp, err := efc.R().Delete(delURL)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.New(string(resp.Body()))
	}
	return nil
}

// Update is used to set the admin or operating state of the device by unique name of the device.
// TODO support to update other fields
func (efc *EdgexDeviceClient) Update(ctx context.Context, device *devicev1alpha1.Device, options edgeCli.UpdateOptions) (*devicev1alpha1.Device, error) {
	actualDeviceName := getEdgeDeviceName(device)
	putURL := fmt.Sprintf("http://%s%s/name/%s", efc.CoreMetaAddr, DevicePath, actualDeviceName)
	if device == nil {
		return nil, nil
	}
	updateData := map[string]string{}
	if device.Spec.AdminState != "" {
		updateData["adminState"] = string(device.Spec.AdminState)
	}
	if device.Spec.OperatingState != "" {
		updateData["operatingState"] = string(device.Spec.OperatingState)
	}
	if len(updateData) == 0 {
		return nil, nil
	}

	data, _ := json.Marshal(updateData)
	rep, err := resty.New().R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Put(putURL)
	if err != nil {
		return nil, err
	} else if rep.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to update device: %s, get response: %s", actualDeviceName, string(rep.Body()))
	}
	return device, nil
}

// Get is used to query the device information corresponding to the device name
func (efc *EdgexDeviceClient) Get(ctx context.Context, deviceName string, options edgeCli.GetOptions) (*devicev1alpha1.Device, error) {
	klog.V(5).Infof("will get Devices: %s", deviceName)
	var device devicev1alpha1.Device
	getURL := fmt.Sprintf("http://%s%s/name/%s", efc.CoreMetaAddr, DevicePath, deviceName)
	resp, err := efc.R().Get(getURL)
	if err != nil {
		return &device, err
	}
	if string(resp.Body()) == "Item not found\n" {
		return &device, errors.New("Item not found")
	}
	var dp models.Device
	err = json.Unmarshal(resp.Body(), &dp)
	device = toKubeDevice(dp)
	return &device, err
}

// List is used to get all device objects on edge platform
// The Hanoi version currently supports only a single label and does not support other filters
func (efc *EdgexDeviceClient) List(ctx context.Context, options edgeCli.ListOptions) ([]devicev1alpha1.Device, error) {
	lp := fmt.Sprintf("http://%s%s", efc.CoreMetaAddr, DevicePath)
	if options.LabelSelector != nil {
		if _, ok := options.LabelSelector["label"]; ok {
			lp = strings.Join([]string{lp, strings.Join([]string{"label", options.LabelSelector["label"]}, "/")}, "/")
		}
	}
	resp, err := efc.R().EnableTrace().Get(lp)
	if err != nil {
		return nil, err
	}
	dps := []models.Device{}
	if err := json.Unmarshal(resp.Body(), &dps); err != nil {
		return nil, err
	}
	var res []devicev1alpha1.Device
	for _, dp := range dps {
		res = append(res, toKubeDevice(dp))
	}
	return res, nil
}

func (efc *EdgexDeviceClient) GetPropertyState(ctx context.Context, propertyName string, d *devicev1alpha1.Device, options edgeCli.GetOptions) (*devicev1alpha1.ActualPropertyState, error) {
	actualDeviceName := getEdgeDeviceName(d)
	// get the old property from status
	oldAps, exist := d.Status.DeviceProperties[propertyName]
	propertyGetURL := ""
	// 1. query the Get URL of an property
	if !exist || (exist && oldAps.GetURL == "") {
		commandRep, err := efc.GetCommandResponseByName(actualDeviceName)
		if err != nil {
			return &devicev1alpha1.ActualPropertyState{}, err
		}
		for _, c := range commandRep.Commands {
			if c.Name == propertyName {
				propertyGetURL = c.Get.URL
				break
			}
		}
		if propertyGetURL == "" {
			return nil, fmt.Errorf("this property %s is not exist", propertyName)
		}
	} else {
		propertyGetURL = oldAps.GetURL
	}
	// 2. get the actual property value by the getURL
	actualPropertyState := devicev1alpha1.ActualPropertyState{
		Name:   propertyName,
		GetURL: propertyGetURL,
	}
	if resp, err := getPropertyState(propertyGetURL); err != nil {
		return nil, err
	} else {
		var event models.Event
		if err := json.Unmarshal(resp.Body(), &event); err != nil {
			return &devicev1alpha1.ActualPropertyState{}, err
		}
		actualPropertyState.ActualValue = getPropertyValueFromEvent(propertyName, event)
	}
	return &actualPropertyState, nil
}

// getPropertyState returns different error messages according to the status code
func getPropertyState(getURL string) (*resty.Response, error) {
	resp, err := resty.New().R().Get(getURL)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode() == 400 {
		err = errors.New("request is in an invalid state")
	} else if resp.StatusCode() == 404 {
		err = errors.New("the requested resource does not exist")
	} else if resp.StatusCode() == 423 {
		err = errors.New("the device is locked (AdminState) or down (OperatingState)")
	} else if resp.StatusCode() == 500 {
		err = errors.New("an unexpected error occurred on the server")
	}
	return resp, err
}

func (efc *EdgexDeviceClient) UpdatePropertyState(ctx context.Context, propertyName string, d *devicev1alpha1.Device, options edgeCli.UpdateOptions) error {
	// Get the actual device name
	acturalDeviceName := getEdgeDeviceName(d)

	dps := d.Spec.DeviceProperties[propertyName]
	if dps.PutURL == "" {
		putURL, err := efc.getPropertyPutURL(acturalDeviceName, dps.Name)
		if err != nil {
			return err
		}
		dps.PutURL = putURL
	}
	// set the device property to desired state
	klog.V(5).Infof("setting the property %s to desired value", dps.Name)
	rep, err := resty.New().R().
		SetHeader("Content-Type", "application/json").
		SetBody([]byte(fmt.Sprintf(`{"%s": "%s"}`, dps.Name, dps.DesiredValue))).
		Put(dps.PutURL)
	if err != nil {
		return err
	} else if rep.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to set property: %s, get response: %s", dps.Name, string(rep.Body()))
	} else if rep.Body() != nil {
		// If the parameters are illegal, such as out of range, the 200 status code is also returned, but the description appears in the body
		a := string(rep.Body())
		if strings.Contains(a, "execWriteCmd") {
			return fmt.Errorf("failed to set property: %s, get response: %s", dps.Name, string(rep.Body()))
		}
	}
	return nil
}

// Gets the putURL from edgex foundry which is used to set the device property's value
func (efc *EdgexDeviceClient) getPropertyPutURL(deviceName, cmdName string) (string, error) {
	cr, err := efc.GetCommandResponseByName(deviceName)
	if err != nil {
		return "", err
	}
	for _, c := range cr.Commands {
		if cmdName == c.Name {
			return c.Put.URL, nil
		}
	}
	return "", errors.New("corresponding command is not found")
}

// ListPropertiesState gets all the actual property information about a device
func (efc *EdgexDeviceClient) ListPropertiesState(ctx context.Context, device *devicev1alpha1.Device, options edgeCli.ListOptions) (map[string]devicev1alpha1.DesiredPropertyState, map[string]devicev1alpha1.ActualPropertyState, error) {
	actualDeviceName := getEdgeDeviceName(device)

	dps := map[string]devicev1alpha1.DesiredPropertyState{}
	aps := map[string]devicev1alpha1.ActualPropertyState{}
	cr, err := efc.GetCommandResponseByName(actualDeviceName)
	if err != nil {
		return dps, aps, err
	}

	for _, c := range cr.Commands {
		// DesiredPropertyState only store the basic information and does not set DesiredValue
		resp, err := getPropertyState(c.Get.URL)
		dps[c.Name] = devicev1alpha1.DesiredPropertyState{Name: c.Name, PutURL: c.Put.URL}
		if err != nil {
			aps[c.Name] = devicev1alpha1.ActualPropertyState{Name: c.Name, GetURL: c.Get.URL}
		} else {
			var event models.Event
			if err := json.Unmarshal(resp.Body(), &event); err != nil {
				klog.V(5).ErrorS(err, "failed to decode the response ", "response", resp)
				continue
			}
			readingName := c.Name
			getResp := c.Get.Responses
			for _, it := range getResp {
				if it.Code == "200" {
					expectValues := it.ExpectedValues
					if len(expectValues) == 1 {
						readingName = expectValues[0]
					}
				}
			}
			klog.V(5).Infof("get reading name %s for command %s of device %s", readingName, c.Name, device.Name)
			actualValue := getPropertyValueFromEvent(readingName, event)

			aps[c.Name] = devicev1alpha1.ActualPropertyState{Name: c.Name, GetURL: c.Get.URL, ActualValue: actualValue}
		}
	}
	return dps, aps, nil
}

// The actual property value is resolved from the returned event
func getPropertyValueFromEvent(propertyName string, modelEvent models.Event) string {
	actualValue := ""
	if len(modelEvent.Readings) == 1 {
		if propertyName == modelEvent.Readings[0].Name {
			actualValue = modelEvent.Readings[0].Value
		}
	} else {
		for _, k := range modelEvent.Readings {
			currentProperty := strings.Join([]string{k.Name, k.Value}, ":")
			if actualValue == "" {
				actualValue = currentProperty
			} else {
				actualValue = strings.Join([]string{actualValue, currentProperty}, ", ")
			}
		}
	}
	return actualValue
}

// GetCommandResponseByName gets all commands supported by the device
func (efc *EdgexDeviceClient) GetCommandResponseByName(deviceName string) (
	models.CommandResponse, error) {
	klog.V(5).Infof("will get CommandResponses of device: %s", deviceName)

	var vd models.CommandResponse
	getURL := fmt.Sprintf("http://%s%s/name/%s", efc.CoreCommandAddr, CommandResponsePath, deviceName)

	resp, err := efc.R().Get(getURL)
	if err != nil {
		return vd, err
	}
	if strings.Contains(string(resp.Body()), "Item not found") {
		return vd, errors.New("Item not found")
	}
	err = json.Unmarshal(resp.Body(), &vd)
	return vd, err
}
