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

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
	edgeCli "github.com/openyurtio/device-controller/clients"
	efCli "github.com/openyurtio/device-controller/clients/edgex-foundry"
	"github.com/openyurtio/device-controller/controllers/util"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

type DeviceSyncer struct {
	// kubernetes client
	client.Client
	// which nodePool deviceController is deployed in
	NodePool string
	// edge platform's client
	deviceCli edgeCli.DeviceInterface
	// syncing period in seconds
	syncPeriod time.Duration
	log        logr.Logger
}

// NewDeviceSyncer initialize a New DeviceSyncer
func NewDeviceSyncer(client client.Client,
	logr logr.Logger,
	periodSecs uint32, cfg *rest.Config) (DeviceSyncer, error) {
	log := logr.WithName("syncer").WithName("Device")
	coreMetaCliInfo := efCli.ClientURL{Host: "edgex-core-metadata", Port: 48081}
	coreCmdCliInfo := efCli.ClientURL{Host: "edgex-core-command", Port: 48082}

	nodePool, err := util.GetNodePool(cfg)
	if err != nil {
		return DeviceSyncer{}, err
	}

	return DeviceSyncer{
		syncPeriod: time.Duration(periodSecs) * time.Second,
		deviceCli:  efCli.NewEdgexDeviceClient(coreMetaCliInfo, coreCmdCliInfo, logr),
		Client:     client,
		log:        log,
		NodePool:   nodePool,
	}, nil
}

// NewDeviceSyncerRunnablel initialize a controller-runtime manager runnable
func (ds *DeviceSyncer) NewDeviceSyncerRunnablel() ctrlmgr.RunnableFunc {
	return func(ctx context.Context) error {
		ds.Run(ctx.Done())
		return nil
	}
}

func (ds *DeviceSyncer) Run(stop <-chan struct{}) {
	ds.log.V(1).Info("starting the DeviceSyncer...")
	go func() {
		for {
			<-time.After(ds.syncPeriod)
			// 1. get device on edge platform and OpenYurt
			edgeDevices, kubeDevices, err := ds.getAllDevices()
			if err != nil {
				ds.log.V(3).Error(err, "fail to list the devices")
				continue
			}

			// 2. find the device that need to be synchronized
			redundantEdgeDevices, redundantKubeDevices, syncedDevices := ds.findDiffDevice(edgeDevices, kubeDevices)
			ds.log.V(1).Info("The number of devices waiting for synchronization",
				"Edge device should be added to OpenYurt", len(redundantEdgeDevices),
				"OpenYurt device that should be deleted", len(redundantKubeDevices),
				"Devices that should be synchronized", len(syncedDevices))

			// 3. create device on OpenYurt which are exists in edge platform but not in OpenYurt
			if err := ds.syncEdgeToKube(redundantEdgeDevices); err != nil {
				ds.log.V(3).Error(err, "fail to create devices on OpenYurt")
				continue
			}

			// 4. delete redundant device on OpenYurt
			if err := ds.deleteDevices(redundantKubeDevices); err != nil {
				ds.log.V(3).Error(err, "fail to delete redundant devices on OpenYurt")
				continue
			}

			// 5. update device status on OpenYurt
			if err := ds.updateDevices(syncedDevices); err != nil {
				ds.log.Error(err, "fail to update devices status")
				continue
			}

			ds.log.V(1).Info("One round of Device synchronization is complete")

		}
	}()

	<-stop
	ds.log.V(1).Info("stopping the device syncer")
}

// Get the existing Device on the Edge platform, as well as OpenYurt existing Device
// edgeDevice：map[actualName]device
// kubeDevice：map[actualName]device
func (ds *DeviceSyncer) getAllDevices() (map[string]devicev1alpha1.Device, map[string]devicev1alpha1.Device, error) {
	edgeDevice := map[string]devicev1alpha1.Device{}
	kubeDevice := map[string]devicev1alpha1.Device{}
	// 1. list devices on edge platform
	eDevs, err := ds.deviceCli.List(nil, edgeCli.ListOptions{})
	if err != nil {
		ds.log.V(4).Error(err, "fail to list the devices object on the Edge Platform")
		return edgeDevice, kubeDevice, err
	}
	// 2. list devices on OpenYurt (filter objects belonging to edgeServer)
	var kDevs devicev1alpha1.DeviceList
	listOptions := client.MatchingFields{"spec.nodePool": ds.NodePool}
	if err = ds.List(context.TODO(), &kDevs, listOptions); err != nil {
		ds.log.V(4).Error(err, "fail to list the devices object on the OpenYurt")
		return edgeDevice, kubeDevice, err
	}
	for i := range eDevs {
		deviceName := util.GetEdgeDeviceName(&eDevs[i], EdgeXObjectName)
		edgeDevice[deviceName] = eDevs[i]
	}

	for i := range kDevs.Items {
		deviceName := util.GetEdgeDeviceName(&kDevs.Items[i], EdgeXObjectName)
		kubeDevice[deviceName] = kDevs.Items[i]
	}
	return edgeDevice, kubeDevice, nil
}

// Get the list of devices that need to be added, deleted and updated
func (ds *DeviceSyncer) findDiffDevice(
	edgeDevice map[string]devicev1alpha1.Device, kubeDevice map[string]devicev1alpha1.Device) (
	redundantEdgeDevices map[string]*devicev1alpha1.Device, redundantKubeDevices map[string]*devicev1alpha1.Device, syncedDevices map[string]*devicev1alpha1.Device) {

	redundantEdgeDevices = map[string]*devicev1alpha1.Device{}
	redundantKubeDevices = map[string]*devicev1alpha1.Device{}
	syncedDevices = map[string]*devicev1alpha1.Device{}

	for n := range edgeDevice {
		tmp := edgeDevice[n]
		edName := util.GetEdgeDeviceName(&tmp, EdgeXObjectName)
		if _, exists := kubeDevice[edName]; !exists {
			ed := edgeDevice[n]
			redundantEdgeDevices[edName] = ds.completeCreateContent(&ed)
		} else {
			kd := kubeDevice[edName]
			ed := edgeDevice[edName]
			syncedDevices[edName] = ds.completeUpdateContent(&kd, &ed)
		}
	}

	for n, v := range kubeDevice {
		if !v.Status.Synced {
			continue
		}
		tmp := kubeDevice[n]
		kdName := util.GetEdgeDeviceName(&tmp, EdgeXObjectName)
		if _, exists := edgeDevice[kdName]; !exists {
			kd := kubeDevice[n]
			redundantKubeDevices[kdName] = &kd
		}
	}
	return
}

// syncEdgeToKube creates device on OpenYurt which are exists in edge platform but not in OpenYurt
func (ds *DeviceSyncer) syncEdgeToKube(edgeDevs map[string]*devicev1alpha1.Device) error {
	for _, ed := range edgeDevs {
		if err := ds.Client.Create(context.TODO(), ed); err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}
			ds.log.V(5).Info("created device failed:",
				"device", strings.ToLower(ed.Name))
			return err
		}
	}
	return nil
}

// deleteDevices deletes redundant device on OpenYurt
func (ds *DeviceSyncer) deleteDevices(redundantKubeDevices map[string]*devicev1alpha1.Device) error {
	for i := range redundantKubeDevices {
		if err := ds.Client.Delete(context.TODO(), redundantKubeDevices[i]); err != nil {
			ds.log.V(5).Error(err, "fail to delete the Device on OpenYurt",
				"device", redundantKubeDevices[i].Name)
			return err
		}
	}
	return nil
}

// updateDevicesStatus updates device status on OpenYurt
func (ds *DeviceSyncer) updateDevices(syncedDevices map[string]*devicev1alpha1.Device) error {
	for n := range syncedDevices {
		if err := ds.Client.Status().Update(context.TODO(), syncedDevices[n]); err != nil {
			if apierrors.IsConflict(err) {
				ds.log.Info("----Conflict")
				continue
			}
			return err
		}
	}
	return nil
}

// completeCreateContent completes the content of the device which will be created on OpenYurt
func (ds *DeviceSyncer) completeCreateContent(edgeDevice *devicev1alpha1.Device) *devicev1alpha1.Device {
	createDevice := edgeDevice.DeepCopy()
	createDevice.Spec.NodePool = ds.NodePool
	createDevice.Name = strings.Join([]string{ds.NodePool, createDevice.Name}, "-")
	createDevice.Spec.Managed = false

	return createDevice
}

// completeUpdateContent completes the content of the device which will be updated on OpenYurt
func (ds *DeviceSyncer) completeUpdateContent(kubeDevice *devicev1alpha1.Device, edgeDevice *devicev1alpha1.Device) *devicev1alpha1.Device {
	updatedDevice := kubeDevice.DeepCopy()
	_, aps, _ := ds.deviceCli.ListPropertiesState(nil, updatedDevice, edgeCli.ListOptions{})
	// update device status
	updatedDevice.Status.LastConnected = edgeDevice.Status.LastConnected
	updatedDevice.Status.LastReported = edgeDevice.Status.LastReported
	updatedDevice.Status.AdminState = edgeDevice.Status.AdminState
	updatedDevice.Status.OperatingState = edgeDevice.Status.OperatingState
	updatedDevice.Status.DeviceProperties = aps
	return updatedDevice
}
