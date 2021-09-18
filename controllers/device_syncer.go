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

package controllers

import (
	"context"
	"strings"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	devv1 "github.com/openyurtio/device-controller/api/v1alpha1"
	coremetacli "github.com/openyurtio/device-controller/clients/core-metadata"
)

type DeviceSyncer struct {
	// syncing period in seconds
	syncPeriod time.Duration
	// EdgeX core-data-service's client
	*coremetacli.CoreMetaClient
	// Kubernetes client
	client.Client
	log logr.Logger
}

// NewDeviceSyncerRunnablel initialize a controller-runtime manager runnable
func (ds *DeviceSyncer) NewDeviceSyncerRunnablel() ctrlmgr.RunnableFunc {
	return func(ctx context.Context) error {
		ds.Run(ctx.Done())
		return nil
	}
}

func (ds *DeviceSyncer) Run(stop <-chan struct{}) {
	ds.log.Info("starting the DeviceSyncer...")
	go func() {
		for {
			<-time.After(ds.syncPeriod)
			// list devices on edgex foundry
			eDevs, err := ds.ListDevices()
			if err != nil {
				ds.log.Error(err, "fail to list the devices object on the EdgeX Foundry")
				continue
			}
			// list devices on Kubernetes
			var kDevs devv1.DeviceList
			if err := ds.List(context.TODO(), &kDevs); err != nil {
				ds.log.Error(err, "fail to list the devices object on the Kubernetes")
				continue
			}
			// create the devices on Kubernetes but not on EdgeX
			newKDevs := findNewDevices(eDevs, kDevs.Items)
			if len(newKDevs) != 0 {
				if err := createDevices(ds.log, ds.Client, newKDevs); err != nil {
					ds.log.Error(err, "fail to create devices")
					continue
				}
			}
			ds.log.V(5).Info("new devices not found")
		}
	}()

	<-stop
	ds.log.Info("stopping the device syncer")
}

// NewDeviceSyncer initialize a New DeviceSyncer
func NewDeviceSyncer(client client.Client,
	logr logr.Logger,
	periodSecs uint32) DeviceSyncer {
	log := logr.WithName("syncer").WithName("Device")
	return DeviceSyncer{
		syncPeriod: time.Duration(periodSecs) * time.Second,
		CoreMetaClient: coremetacli.NewCoreMetaClient(
			"edgex-core-metadata.default", 48081, log),
		Client: client,
		log:    log,
	}
}

// findNewDevices finds devices that have been created on the EdgeX but
// not the Kubernetes
func findNewDevices(
	edgeXDevs []models.Device,
	kubeDevs []devv1.Device) []models.Device {
	var retDevs []models.Device
	for _, exd := range edgeXDevs {
		var exist bool
		for _, kd := range kubeDevs {
			if strings.ToLower(exd.Name) == kd.Name {
				exist = true
				break
			}
		}
		if !exist {
			retDevs = append(retDevs, exd)
		}
	}

	return retDevs
}

// createDevices creates the list of devices
func createDevices(log logr.Logger, cli client.Client, edgeXDevs []models.Device) error {
	for _, ed := range edgeXDevs {
		kd := toKubeDevice(ed)
		if err := cli.Create(context.TODO(), &kd); err != nil {
			if apierrors.IsAlreadyExists(err) {
				log.Info("Device already exist on Kubernetes",
					"device", strings.ToLower(ed.Name))
				continue
			}
			log.Error(err, "fail to create the Device on Kubernetes",
				"device", ed.Name)
			return err
		}
	}
	return nil
}

// toKubeDevice serialize the EdgeX Device to the corresponding Kubernetes Device
func toKubeDevice(ed models.Device) devv1.Device {
	var loc string
	if ed.Location != nil {
		loc = ed.Location.(string)
	}
	return devv1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ed.Name),
			Namespace: "default",
			Labels: map[string]string{
				EdgeXObjectName: ed.Name,
			},
		},
		Spec: devv1.DeviceSpec{
			Description:    ed.Description,
			AdminState:     toKubeAdminState(ed.AdminState),
			OperatingState: toKubeOperatingState(ed.OperatingState),
			Protocols:      toKubeProtocols(ed.Protocols),
			Labels:         ed.Labels,
			Location:       loc,
			Service:        ed.Service.Name,
			Profile:        ed.Profile.Name,
		},
		Status: devv1.DeviceStatus{
			LastConnected: ed.LastConnected,
			LastReported:  ed.LastReported,
			AddedToEdgeX:  true,
			Id:            ed.Id,
		},
	}
}

// toKubeDevice serialize the EdgeX AdminState to the corresponding Kubernetes AdminState
func toKubeAdminState(ea models.AdminState) devv1.AdminState {
	if ea == models.Locked {
		return devv1.Locked
	}
	return devv1.UnLocked
}

// toKubeDevice serialize the EdgeX OperatingState to the corresponding
// Kubernetes OperatingState
func toKubeOperatingState(ea models.OperatingState) devv1.OperatingState {
	if ea == models.Enabled {
		return devv1.Enabled
	}
	return devv1.Disabled
}

// toKubeProtocols serialize the EdgeX ProtocolProperties to the corresponding
// Kubernetes OperatingState
func toKubeProtocols(
	eps map[string]models.ProtocolProperties) map[string]devv1.ProtocolProperties {
	ret := map[string]devv1.ProtocolProperties{}
	for k, v := range eps {
		ret[k] = devv1.ProtocolProperties(v)
	}
	return ret
}
