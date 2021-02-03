package core_metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
)

type CoreMetaClient struct {
	*resty.Client
	Host string
	Port int
	logr.Logger
}

const (
	DeviceProfilePath = "/api/v1/deviceprofile"
	DeviceServicePath = "/api/v1/deviceservice"
	DevicePath        = "/api/v1/device"
	AddressablePath   = "/api/v1/addressable"
)

func NewCoreMetaClient(host string, port int, log logr.Logger) *CoreMetaClient {
	return &CoreMetaClient{
		Client: resty.New(),
		Host:   host,
		Port:   port,
		Logger: log,
	}
}

func (cdc *CoreMetaClient) ListDeviceProfile() (
	[]models.DeviceProfile, error) {
	cdc.V(5).Info("will list DeviceProfiles")
	lp := fmt.Sprintf("http://%s:%d%s",
		cdc.Host, cdc.Port, DeviceProfilePath)
	resp, err := cdc.R().
		EnableTrace().
		Get(lp)
	if err != nil {
		return nil, err
	}
	dps := []models.DeviceProfile{}
	if err := json.Unmarshal(resp.Body(), &dps); err != nil {
		return nil, err
	}
	return dps, nil
}

func (cdc *CoreMetaClient) GetDeviceProfileByName(name string) (
	models.DeviceProfile, error) {
	cdc.V(5).Info("will get DeviceProfiles",
		"DeviceProfile", name)
	var dp models.DeviceProfile
	getURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		cdc.Host, cdc.Port, DeviceProfilePath, name)
	resp, err := cdc.R().Get(getURL)
	if err != nil {
		return dp, err
	}
	if string(resp.Body()) == "Item not found\n" {
		return dp, errors.New("Item not found")
	}
	err = json.Unmarshal(resp.Body(), &dp)
	return dp, err
}

func (cdc *CoreMetaClient) GetDeviceProfilesByLabel(label string) (
	[]models.DeviceProfile, error) {
	panic("NOT IMPLEMENT YET")
}

func (cdc *CoreMetaClient) AddDeviceProfile(dp models.DeviceProfile) (
	string, error) {
	cdc.V(5).Info("will add the DeviceProfiles",
		"DeviceProfile", dp.Name)
	dpJson, err := json.Marshal(&dp)
	if err != nil {
		return "", err
	}
	postPath := fmt.Sprintf("http://%s:%d%s",
		cdc.Host, cdc.Port, DeviceProfilePath)
	resp, err := cdc.R().
		SetBody(dpJson).Post(postPath)
	if err != nil {
		return "", err
	}
	return string(resp.Body()), err
}

func (cdc *CoreMetaClient) DeleteDeviceProfileByName(name string) error {
	cdc.V(5).Info("will delete the DeviceProfile",
		"DeviceProfile", name)
	delURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		cdc.Host, cdc.Port, DeviceProfilePath, name)
	resp, err := cdc.R().Delete(delURL)
	if err != nil {
		return err
	}
	if string(resp.Body()) != "true" {
		return errors.New(string(resp.Body()))
	}
	return nil
}

func (cdc *CoreMetaClient) ListDevices() (
	[]models.Device, error) {
	cdc.V(5).Info("will list Devices")
	lp := fmt.Sprintf("http://%s:%d%s",
		cdc.Host, cdc.Port, DevicePath)
	resp, err := cdc.R().
		EnableTrace().
		Get(lp)
	if err != nil {
		return nil, err
	}
	dps := []models.Device{}
	if err := json.Unmarshal(resp.Body(), &dps); err != nil {
		return nil, err
	}
	return dps, nil
}

func (cdc *CoreMetaClient) GetDeviceByName(name string) (
	models.Device, error) {
	cdc.V(5).Info("will get Devices",
		"Device", name)
	var dp models.Device
	getURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		cdc.Host, cdc.Port, DevicePath, name)
	resp, err := cdc.R().Get(getURL)
	if err != nil {
		return dp, err
	}
	if string(resp.Body()) == "Item not found\n" {
		return dp, errors.New("Item not found")
	}
	err = json.Unmarshal(resp.Body(), &dp)
	return dp, err
}

func (cdc *CoreMetaClient) GetDevicesByLabel(label string) (
	[]models.Device, error) {
	panic("NOT IMPLEMENT YET")
}

func (cdc *CoreMetaClient) AddDevice(dp models.Device) (
	string, error) {
	cdc.V(5).Info("will add the Devices",
		"Device", dp.Name)
	dpJson, err := json.Marshal(&dp)
	if err != nil {
		return "", err
	}
	postPath := fmt.Sprintf("http://%s:%d%s",
		cdc.Host, cdc.Port, DevicePath)
	resp, err := cdc.R().
		SetBody(dpJson).Post(postPath)
	if err != nil {
		return "", err
	}
	if strings.Contains(string(resp.Body()), "no item found") {
		return "", errors.New("Item not found")
	}
	return string(resp.Body()), err
}

func (cdc *CoreMetaClient) DeleteDeviceByName(name string) error {
	cdc.V(5).Info("will delete the Device",
		"Device", name)
	delURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		cdc.Host, cdc.Port, DevicePath, name)
	resp, err := cdc.R().Delete(delURL)
	if err != nil {
		return err
	}
	if string(resp.Body()) != "true" {
		return errors.New(string(resp.Body()))
	}
	return nil
}

func (cdc *CoreMetaClient) ListDeviceServices() (
	[]models.DeviceService, error) {
	cdc.V(5).Info("will list DeviceServices")
	lp := fmt.Sprintf("http://%s:%d%s",
		cdc.Host, cdc.Port, DeviceServicePath)
	resp, err := cdc.R().
		EnableTrace().
		Get(lp)
	if err != nil {
		return nil, err
	}
	dps := []models.DeviceService{}
	if err := json.Unmarshal(resp.Body(), &dps); err != nil {
		return nil, err
	}
	return dps, nil
}

func (cdc *CoreMetaClient) GetDeviceServiceByName(name string) (
	models.DeviceService, error) {
	cdc.V(5).Info("will get DeviceServices",
		"DeviceService", name)
	var ds models.DeviceService
	getURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		cdc.Host, cdc.Port, DeviceServicePath, name)
	resp, err := cdc.R().Get(getURL)
	if err != nil {
		return ds, err
	}
	if string(resp.Body()) == "Item not found\n" ||
		strings.HasPrefix(string(resp.Body()), "no item found") {
		return ds, errors.New("Item not found")
	}
	err = json.Unmarshal(resp.Body(), &ds)
	return ds, err
}

func (cdc *CoreMetaClient) GetDeviceServicesByLabel(label string) (
	[]models.DeviceService, error) {
	panic("NOT IMPLEMENT YET")
}

func (cdc *CoreMetaClient) AddDeviceService(dp models.DeviceService) (
	string, error) {
	cdc.V(5).Info("will add the DeviceServices",
		"DeviceService", dp.Name)
	dpJson, err := json.Marshal(&dp)
	if err != nil {
		return "", err
	}
	postPath := fmt.Sprintf("http://%s:%d%s",
		cdc.Host, cdc.Port, DeviceServicePath)
	resp, err := cdc.R().
		SetBody(dpJson).Post(postPath)
	if err != nil {
		return "", err
	}
	return string(resp.Body()), err
}

func (cdc *CoreMetaClient) DeleteDeviceServiceByName(name string) error {
	cdc.V(5).Info("will delete the DeviceService",
		"DeviceService", name)
	delURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		cdc.Host, cdc.Port, DeviceServicePath, name)
	resp, err := cdc.R().Delete(delURL)
	if err != nil {
		return err
	}
	if string(resp.Body()) != "true" {
		return errors.New(string(resp.Body()))
	}
	return nil
}

func (cdc *CoreMetaClient) ListAddressables() (
	[]models.Addressable, error) {
	cdc.V(5).Info("will list Addressables")
	lp := fmt.Sprintf("http://%s:%d%s",
		cdc.Host, cdc.Port, AddressablePath)
	resp, err := cdc.R().
		EnableTrace().
		Get(lp)
	if err != nil {
		return nil, err
	}
	dps := []models.Addressable{}
	if err := json.Unmarshal(resp.Body(), &dps); err != nil {
		return nil, err
	}
	return dps, nil
}

func (cdc *CoreMetaClient) GetAddressableByName(name string) (
	models.Addressable, error) {
	cdc.V(5).Info("will get Addressables",
		"Addressable", name)
	var dp models.Addressable
	getURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		cdc.Host, cdc.Port, AddressablePath, name)
	resp, err := cdc.R().Get(getURL)
	if err != nil {
		return dp, err
	}
	if string(resp.Body()) == "Item not found\n" {
		return dp, errors.New("Item not found")
	}
	err = json.Unmarshal(resp.Body(), &dp)
	return dp, err
}

func (cdc *CoreMetaClient) GetAddressablesByLabel(label string) (
	[]models.Addressable, error) {
	panic("NOT IMPLEMENT YET")
}

func (cdc *CoreMetaClient) AddAddressable(dp models.Addressable) (
	string, error) {
	cdc.V(5).Info("will add the Addressables",
		"Addressable", dp.Name)
	dpJson, err := json.Marshal(&dp)
	if err != nil {
		return "", err
	}
	postPath := fmt.Sprintf("http://%s:%d%s",
		cdc.Host, cdc.Port, AddressablePath)
	resp, err := cdc.R().
		SetBody(dpJson).Post(postPath)
	if err != nil {
		return "", err
	}
	return string(resp.Body()), err
}

func (cdc *CoreMetaClient) DeleteAddressableByName(name string) error {
	cdc.V(5).Info("will delete the Addressable",
		"Addressable", name)
	delURL := fmt.Sprintf("http://%s:%d%s/name/%s",
		cdc.Host, cdc.Port, AddressablePath, name)
	resp, err := cdc.R().Delete(delURL)
	if err != nil {
		return err
	}
	if string(resp.Body()) != "true" {
		return errors.New(string(resp.Body()))
	}
	return nil
}
