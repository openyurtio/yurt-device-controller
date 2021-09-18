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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/go-logr/logr"
	devv1 "github.com/openyurtio/device-controller/api/v1alpha1"
	coremetacli "github.com/openyurtio/device-controller/clients/core-metadata"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeviceServiceSyncer struct {
	// syncing period in seconds
	syncPeriod time.Duration
	// EdgeX core-data-service's client
	*coremetacli.CoreMetaClient
	// Kubernetes client
	client.Client
	log logr.Logger
}

func NewDeviceServiceSyncer(client client.Client,
	logr logr.Logger,
	periodSecs uint32) DeviceServiceSyncer {
	log := logr.WithName("syncer").WithName("DeviceService")
	return DeviceServiceSyncer{
		syncPeriod: time.Duration(periodSecs) * time.Second,
		CoreMetaClient: coremetacli.NewCoreMetaClient(
			"edgex-core-metadata.default", 48081, log),
		Client: client,
		log:    log,
	}
}

func (dss *DeviceServiceSyncer) NewDeviceServiceSyncerRunnable() ctrlmgr.RunnableFunc {
	return func(ctx context.Context) error {
		dss.Run(ctx.Done())
		return nil
	}
}

func (ds *DeviceServiceSyncer) Run(stop <-chan struct{}) {
	ds.log.Info("starting the DeviceServiceSyncer...")
	go func() {
		for {
			<-time.After(ds.syncPeriod)
			// list deviceservice on edgex foundry
			eDevs, err := ds.ListDeviceServices()
			if err != nil {
				ds.log.Error(err, "fail to list the deviceservice object on the EdgeX Foundry")
				continue
			}
			// list deviceservice on Kubernetes
			var kDevs devv1.DeviceServiceList
			if err := ds.List(context.TODO(), &kDevs); err != nil {
				ds.log.Error(err, "fail to list the deviceservice object on the Kubernetes")
				continue
			}
			// create the deviceservice on Kubernetes but not on EdgeX
			newKDevs := findNewDeviceService(eDevs, kDevs.Items)
			if len(newKDevs) != 0 {
				if err := createDeviceService(ds.log, ds.Client, newKDevs); err != nil {
					ds.log.Error(err, "fail to create deviceservice")
					continue
				}
			}
			ds.log.V(5).Info("new deviceservice not found")
		}
	}()

	<-stop
	ds.log.Info("stopping the device syncer")
}

// findNewDeviceService finds devices that have been created on the EdgeX but
// not the Kubernetes
func findNewDeviceService(
	edgeXDevs []models.DeviceService,
	kubeDevs []devv1.DeviceService) []models.DeviceService {
	var retDevs []models.DeviceService
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

// createDeviceService creates the list of devices
func createDeviceService(log logr.Logger, cli client.Client, edgeXDevs []models.DeviceService) error {
	for _, ed := range edgeXDevs {
		kd := toKubeDeviceService(ed)
		if err := cli.Create(context.TODO(), &kd); err != nil {
			if apierrors.IsAlreadyExists(err) {
				log.Info("DeviceService already exist on Kubernetes",
					"deviceservice", strings.ToLower(ed.Name))
				continue
			}
			log.Error(err, "fail to create the DeviceService on Kubernetes",
				"deviceservice", ed.Name)
			return err
		}
	}
	return nil
}

func toKubeDeviceService(ds models.DeviceService) devv1.DeviceService {
	return devv1.DeviceService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ds.Name),
			Namespace: "default",
			Labels: map[string]string{
				EdgeXObjectName: ds.Name,
			},
		},
		Spec: devv1.DeviceServiceSpec{
			Description:    ds.Description,
			Id:             ds.Id,
			LastConnected:  ds.LastConnected,
			LastReported:   ds.LastReported,
			OperatingState: toKubeOperatingState(ds.OperatingState),
			Labels:         ds.Labels,
			Addressable:    toKubeAddressable(ds.Addressable),
			AdminState:     toKubeAdminState(ds.AdminState),
		},
	}
}

func toKubeAddressable(ad models.Addressable) devv1.Addressable {
	return devv1.Addressable{
		Id:         ad.Id,
		Name:       ad.Name,
		Protocol:   ad.Protocol,
		HTTPMethod: ad.HTTPMethod,
		Address:    ad.Address,
		Port:       ad.Port,
		Path:       ad.Path,
		Publisher:  ad.Publisher,
		User:       ad.User,
		Password:   ad.Password,
		Topic:      ad.Topic,
	}
}
