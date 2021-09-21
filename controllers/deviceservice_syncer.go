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
	"github.com/openyurtio/device-controller/controllers/util"
	"k8s.io/client-go/rest"
	"strings"
	"time"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
	iotcli "github.com/openyurtio/device-controller/clients"
	edgexCli "github.com/openyurtio/device-controller/clients/edgex-foundry"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

type DeviceServiceSyncer struct {
	// Kubernetes client
	client.Client
	// syncing period in seconds
	syncPeriod       time.Duration
	deviceServiceCli iotcli.DeviceServiceInterface
	log              logr.Logger
	NodePool         string
}

func NewDeviceServiceSyncer(client client.Client,
	logr logr.Logger,
	periodSecs uint32, cfg *rest.Config) (DeviceServiceSyncer, error) {
	log := logr.WithName("syncer").WithName("DeviceService")
	coreMetaCliInfo := edgexCli.ClientURL{Host: "edgex-core-metadata", Port: 48081}

	nodePool, err := util.GetNodePool(cfg)
	if err != nil {
		return DeviceServiceSyncer{}, err
	}
	return DeviceServiceSyncer{
		syncPeriod:       time.Duration(periodSecs) * time.Second,
		deviceServiceCli: edgexCli.NewEdgexDeviceServiceClient(coreMetaCliInfo, logr),
		Client:           client,
		log:              log,
		NodePool:         nodePool,
	}, nil
}

func (ds *DeviceServiceSyncer) NewDeviceServiceSyncerRunnable() ctrlmgr.RunnableFunc {
	return func(ctx context.Context) error {
		ds.Run(ctx.Done())
		return nil
	}
}

func (ds *DeviceServiceSyncer) Run(stop <-chan struct{}) {
	ds.log.V(1).Info("starting the DeviceServiceSyncer...")
	go func() {
		for {
			<-time.After(ds.syncPeriod)
			// 1. get deviceServices on edge platform and OpenYurt
			edgeDeviceServices, kubeDeviceServices, err := ds.getAllDeviceServices()
			if err != nil {
				ds.log.V(3).Error(err, "fail to list the deviceServices")
				continue
			}

			// 2. find the deviceServices that need to be synchronized
			redundantEdgeDeviceServices, redundantKubeDeviceServices, syncedDeviceServices :=
				ds.findDiffDeviceServices(edgeDeviceServices, kubeDeviceServices)
			ds.log.V(1).Info("The number of deviceServices waiting for synchronization",
				"Edge deviceServices should be added to OpenYurt", len(redundantEdgeDeviceServices),
				"OpenYurt deviceServices that should be deleted", len(redundantKubeDeviceServices),
				"DeviceServices that should be synchronized", len(syncedDeviceServices))

			// 3. create deviceServices on OpenYurt which are exists in edge platform but not in OpenYurt
			if err := ds.syncEdgeToKube(redundantEdgeDeviceServices); err != nil {
				ds.log.V(3).Error(err, "fail to create deviceServices on OpenYurt")
				continue
			}

			// 4. delete redundant deviceServices on OpenYurt
			if err := ds.deleteDeviceServices(redundantKubeDeviceServices); err != nil {
				ds.log.V(3).Error(err, "fail to delete redundant deviceServices on OpenYurt")
				continue
			}

			// 5. update deviceService status on OpenYurt
			if err := ds.updateDeviceServices(syncedDeviceServices); err != nil {
				ds.log.Error(err, "fail to update deviceServices")
				continue
			}

			ds.log.V(1).Info("One round of DeviceService synchronization is complete")
		}
	}()

	<-stop
	ds.log.V(1).Info("stopping the deviceService syncer")
}

// Get the existing DeviceService on the Edge platform, as well as OpenYurt existing DeviceService
// edgeDeviceServices：map[actualName]DeviceService
// kubeDeviceServices：map[actualName]DeviceService
func (ds *DeviceServiceSyncer) getAllDeviceServices() (
	map[string]devicev1alpha1.DeviceService, map[string]devicev1alpha1.DeviceService, error) {

	edgeDeviceServices := map[string]devicev1alpha1.DeviceService{}
	kubeDeviceServices := map[string]devicev1alpha1.DeviceService{}

	// 1. list deviceServices on edge platform
	eDevSs, err := ds.deviceServiceCli.List(nil, iotcli.ListOptions{})
	if err != nil {
		ds.log.V(4).Error(err, "fail to list the deviceServices object on the edge platform")
		return edgeDeviceServices, kubeDeviceServices, err
	}
	// 2. list deviceServices on OpenYurt (filter objects belonging to edgeServer)
	var kDevSs devicev1alpha1.DeviceServiceList
	listOptions := client.MatchingFields{"spec.nodePool": ds.NodePool}
	if err = ds.List(context.TODO(), &kDevSs, listOptions); err != nil {
		ds.log.V(4).Error(err, "fail to list the deviceServices object on the Kubernetes")
		return edgeDeviceServices, kubeDeviceServices, err
	}
	for i := range eDevSs {
		deviceServicesName := eDevSs[i].Labels[EdgeXObjectName]
		edgeDeviceServices[deviceServicesName] = eDevSs[i]
	}

	for i := range kDevSs.Items {
		deviceServicesName := kDevSs.Items[i].Labels[EdgeXObjectName]
		kubeDeviceServices[deviceServicesName] = kDevSs.Items[i]
	}
	return edgeDeviceServices, kubeDeviceServices, nil
}

// Get the list of deviceServices that need to be added, deleted and updated
func (ds *DeviceServiceSyncer) findDiffDeviceServices(
	edgeDeviceService map[string]devicev1alpha1.DeviceService, kubeDeviceService map[string]devicev1alpha1.DeviceService) (
	redundantEdgeDeviceServices map[string]*devicev1alpha1.DeviceService, redundantKubeDeviceServices map[string]*devicev1alpha1.DeviceService, syncedDeviceServices map[string]*devicev1alpha1.DeviceService) {

	redundantEdgeDeviceServices = map[string]*devicev1alpha1.DeviceService{}
	redundantKubeDeviceServices = map[string]*devicev1alpha1.DeviceService{}
	syncedDeviceServices = map[string]*devicev1alpha1.DeviceService{}

	for n, v := range edgeDeviceService {
		edName := v.Labels[EdgeXObjectName]
		if _, exists := kubeDeviceService[edName]; !exists {
			ed := edgeDeviceService[n]
			redundantEdgeDeviceServices[edName] = ds.completeCreateContent(&ed)
		} else {
			kd := kubeDeviceService[edName]
			ed := edgeDeviceService[n]
			syncedDeviceServices[edName] = ds.completeUpdateContent(&kd, &ed)
		}
	}

	for k, v := range kubeDeviceService {
		if !v.Status.Synced {
			continue
		}
		kdName := v.Labels[EdgeXObjectName]
		if _, exists := edgeDeviceService[kdName]; !exists {
			kd := kubeDeviceService[k]
			redundantKubeDeviceServices[kdName] = &kd
		}
	}
	return
}

// syncEdgeToKube creates deviceServices on OpenYurt which are exists in edge platform but not in OpenYurt
func (ds *DeviceServiceSyncer) syncEdgeToKube(edgeDevs map[string]*devicev1alpha1.DeviceService) error {
	for _, ed := range edgeDevs {
		if err := ds.Client.Create(context.TODO(), ed); err != nil {
			if apierrors.IsAlreadyExists(err) {
				ds.log.V(5).Info("DeviceService already exist on Kubernetes",
					"DeviceService", strings.ToLower(ed.Name))
				continue
			}
			ds.log.Info("created deviceService failed:",
				"DeviceService", strings.ToLower(ed.Name))
			return err
		}
	}
	return nil
}

// deleteDeviceServices deletes redundant deviceServices on OpenYurt
func (ds *DeviceServiceSyncer) deleteDeviceServices(redundantKubeDeviceServices map[string]*devicev1alpha1.DeviceService) error {
	for i := range redundantKubeDeviceServices {
		if err := ds.Client.Delete(context.TODO(), redundantKubeDeviceServices[i]); err != nil {
			ds.log.V(5).Error(err, "fail to delete the DeviceService on Kubernetes",
				"DeviceService", redundantKubeDeviceServices[i].Name)
			return err
		}
	}
	return nil
}

// updateDeviceServicesStatus updates deviceServices status on OpenYurt
func (ds *DeviceServiceSyncer) updateDeviceServices(syncedDeviceServices map[string]*devicev1alpha1.DeviceService) error {
	for _, sd := range syncedDeviceServices {
		if sd.ObjectMeta.ResourceVersion == "" {
			continue
		}
		if err := ds.Client.Status().Update(context.TODO(), sd); err != nil {
			if apierrors.IsConflict(err) {
				ds.log.V(5).Info("update Conflicts",
					"DeviceService", sd.Name)
				continue
			}
			ds.log.V(5).Error(err, "fail to update the DeviceService on Kubernetes",
				"DeviceService", sd.Name)
			return err
		}
	}
	return nil
}

// completeCreateContent completes the content of the deviceService which will be created on OpenYurt
func (ds *DeviceServiceSyncer) completeCreateContent(edgeDS *devicev1alpha1.DeviceService) *devicev1alpha1.DeviceService {
	createDevice := edgeDS.DeepCopy()
	createDevice.Spec.NodePool = ds.NodePool
	createDevice.Name = strings.Join([]string{ds.NodePool, createDevice.Name}, "-")
	createDevice.Spec.Managed = false
	return createDevice
}

// completeUpdateContent completes the content of the deviceService which will be updated on OpenYurt
func (ds *DeviceServiceSyncer) completeUpdateContent(kubeDS *devicev1alpha1.DeviceService, edgeDS *devicev1alpha1.DeviceService) *devicev1alpha1.DeviceService {
	updatedDS := kubeDS.DeepCopy()
	// update device status
	updatedDS.Status.LastConnected = edgeDS.Status.LastConnected
	updatedDS.Status.LastReported = edgeDS.Status.LastReported
	updatedDS.Status.AdminState = edgeDS.Status.AdminState
	return updatedDS
}
