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

package core_command

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
)

type CoreCommandClient struct {
	*resty.Client
	Host string
	Port int
	logr.Logger
}

const (
	CommandResponsePath = "/api/v1/device"
)

func NewCoreCommandClient(host string, port int, log logr.Logger) *CoreCommandClient {
	return &CoreCommandClient{
		Client: resty.New(),
		Host:   host,
		Port:   port,
		Logger: log,
	}
}

func (cdc *CoreCommandClient) ListCommandResponse() (
	[]models.CommandResponse, error) {
	cdc.Info("will list CommandResponses")
	lp := fmt.Sprintf("http://%s:%d%s",
		cdc.Host, cdc.Port, CommandResponsePath)
	resp, err := cdc.R().
		EnableTrace().
		Get(lp)
	if err != nil {
		return nil, err
	}
	vds := []models.CommandResponse{}
	if err := json.Unmarshal(resp.Body(), &vds); err != nil {
		return nil, err
	}
	return vds, nil
}

func (cdc *CoreCommandClient) GetCommandResponseByName(name string) (
	models.CommandResponse, error) {
	cdc.Info("will get CommandResponses",
		"CommandResponse", name)
	var vd models.CommandResponse
	getURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		cdc.Host, cdc.Port, CommandResponsePath, name)
	resp, err := cdc.R().Get(getURL)
	if err != nil {
		return vd, err
	}
	cdc.Info("---------------", "name", name, "respbody", string(resp.Body()))
	if strings.Contains(string(resp.Body()), "Item not found") {
		return vd, errors.New("Item not found")
	}
	err = json.Unmarshal(resp.Body(), &vd)
	return vd, err
}
