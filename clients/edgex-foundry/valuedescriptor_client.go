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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-resty/resty/v2"
	"k8s.io/klog/v2"
)

type EdgexValueDescriptorClient struct {
	*resty.Client
	CoreDataAddr string
}

const (
	ValueDescriptorPath = "/api/v1/valuedescriptor"
)

func NewValueDescriptorClient(address string) *EdgexValueDescriptorClient {
	return &EdgexValueDescriptorClient{
		Client:       resty.New(),
		CoreDataAddr: address,
	}
}

func (cdc *EdgexValueDescriptorClient) ListValueDescriptor() ([]models.ValueDescriptor, error) {
	klog.V(5).Info("will list ValueDescriptors")
	lp := fmt.Sprintf("http://%s%s", cdc.CoreDataAddr, ValueDescriptorPath)
	resp, err := cdc.R().
		EnableTrace().
		Get(lp)
	if err != nil {
		return nil, err
	}
	vds := []models.ValueDescriptor{}
	if err := json.Unmarshal(resp.Body(), &vds); err != nil {
		return nil, err
	}
	return vds, nil
}

func (cdc *EdgexValueDescriptorClient) GetValueDescriptorByName(name string) (
	models.ValueDescriptor, error) {
	klog.V(5).InfoS("will get ValueDescriptors", "valuedescriptor", name)
	var vd models.ValueDescriptor
	getURL := fmt.Sprintf("http://%s%s/name/%s", cdc.CoreDataAddr, ValueDescriptorPath, name)
	resp, err := cdc.R().Get(getURL)
	if err != nil {
		return vd, err
	}
	if string(resp.Body()) == "Item not found\n" {
		return vd, errors.New("Item not found")
	}
	err = json.Unmarshal(resp.Body(), &vd)
	return vd, err
}

func (cdc *EdgexValueDescriptorClient) GetValueDescriptsByLabel(label string) ([]models.ValueDescriptor, error) {
	panic("NOT IMPLEMENT YET")
}

func (cdc *EdgexValueDescriptorClient) AddValueDescript(vd models.ValueDescriptor) (
	string, error) {
	klog.V(5).InfoS("will add the ValueDescriptors", "valuedescriptor", vd.Name)
	vdJson, err := json.Marshal(&vd)
	if err != nil {
		return "", err
	}
	postPath := fmt.Sprintf("http://%s%s", cdc.CoreDataAddr, ValueDescriptorPath)
	resp, err := cdc.R().
		SetBody(vdJson).Post(postPath)
	if err != nil {
		return "", err
	}
	return string(resp.Body()), err
}

func (cdc *EdgexValueDescriptorClient) DeleteValueDescriptorByName(name string) error {
	klog.V(5).InfoS("will delete the ValueDescriptor", "valuedescriptor", name)
	delURL := fmt.Sprintf("http://%s%s/name/%s", cdc.CoreDataAddr, ValueDescriptorPath, name)
	resp, err := cdc.R().Delete(delURL)
	if err != nil {
		return err
	}
	if string(resp.Body()) != "true" {
		return errors.New(string(resp.Body()))
	}
	return nil
}
