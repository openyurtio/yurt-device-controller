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
	"fmt"
	"reflect"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
	devcli "github.com/openyurtio/device-controller/clients"
	edgexclis "github.com/openyurtio/device-controller/clients/edgex-foundry"
	"github.com/openyurtio/device-controller/cmd/yurt-device-controller/options"
	"github.com/openyurtio/device-controller/controllers/util"
)

type DeviceProfileSyncer struct {
	// syncing period in seconds
	syncPeriod time.Duration
	// EdgeX core-data-service's client
	edgeClient devcli.DeviceProfileInterface
	// Kubernetes client
	client.Client
	NodePool  string
	Namespace string
}

// NewDeviceProfileSyncer initialize a New DeviceProfileSyncer
func NewDeviceProfileSyncer(client client.Client, opts *options.YurtDeviceControllerOptions) (DeviceProfileSyncer, error) {
	return DeviceProfileSyncer{
		syncPeriod: time.Duration(opts.EdgeSyncPeriod) * time.Second,
		edgeClient: edgexclis.NewEdgexDeviceProfile(opts.CoreMetadataAddr),
		Client:     client,
		NodePool:   opts.Nodepool,
		Namespace:  opts.Namespace,
	}, nil
}

// NewDeviceProfileSyncerRunnable initialize a controller-runtime manager runnable
func (dps *DeviceProfileSyncer) NewDeviceProfileSyncerRunnable() ctrlmgr.RunnableFunc {
	return func(ctx context.Context) error {
		dps.Run(ctx.Done())
		return nil
	}
}

func (ds *DeviceProfileSyncer) Run(stop <-chan struct{}) {
	klog.V(1).Info("[DeviceProfile] Starting the syncer...")
	go func() {
		for {
			<-time.After(ds.syncPeriod)
			klog.V(2).Info("[DeviceProfile] Start a round of synchronization.")
			// list devices on edgex foundry
			eDevs, err := ds.edgeClient.List(context.Background(), devcli.ListOptions{})
			if err != nil {
				klog.V(3).ErrorS(err, "fail to list the deviceprofile object on the edge platform")
				continue
			}
			addNodePoolField(eDevs, ds.NodePool)
			// list devices on Kubernetes
			var kDevs devicev1alpha1.DeviceProfileList
			listOptions := client.MatchingFields{"spec.nodePool": ds.NodePool}
			if err := ds.List(context.TODO(), &kDevs, listOptions, client.InNamespace(ds.Namespace)); err != nil {
				klog.V(3).ErrorS(err, "fail to list the deviceprofile object on the Kubernetes")
				continue
			}
			// create the device profiles on Kubernetes but not on EdgeX
			newKDevs, updateKDevs := ds.findNewUpdateDeviceProfile(eDevs, kDevs.Items)
			if len(newKDevs) != 0 {
				if err := createDeviceProfile(ds.Client, newKDevs, ds.NodePool); err != nil {
					klog.V(3).ErrorS(err, "fail to create device profiles")
					continue
				}
			}
			// update the device profiles according EdgeX
			if len(updateKDevs) != 0 {
				// TODO
			}
			// delete the device profiles on Kubernetes but not on Egdex
			deleteKDevs := findDeleteDeviceProfile(eDevs, kDevs.Items)
			if len(deleteKDevs) != 0 {
				if err := deleteDeviceProfile(ds.Client, deleteKDevs); err != nil {
					klog.V(3).ErrorS(err, "fail to delete device profiles")
				}
			}
			klog.V(2).Info("[DeviceProfile] One round of synchronization is complete")
		}
	}()

	<-stop
	klog.V(1).Info("[DeviceProfile] Stopping the syncer")
}

func addNodePoolField(edgeXDevs []devicev1alpha1.DeviceProfile, NodePoolName string) {
	for i := range edgeXDevs {
		edgeXDevs[i].Spec.NodePool = NodePoolName
	}
}

// findNewUpdateDeviceProfile finds deviceprofiles that have been created on the EdgeX but not the Kubernetes
func (ds *DeviceProfileSyncer) findNewUpdateDeviceProfile(edgeXDevs, kubeDevs []devicev1alpha1.DeviceProfile) ([]devicev1alpha1.DeviceProfile, []devicev1alpha1.DeviceProfile) {
	var addDevs, updateDevs []devicev1alpha1.DeviceProfile
	for _, exd := range edgeXDevs {
		var exist bool
		for i, kd := range kubeDevs {
			dp := kubeDevs[i]
			if exd.Name == strings.ToLower(util.GetEdgeDeviceProfileName(&dp, EdgeXObjectName)) {
				exist = true
				if !reflect.DeepEqual(exd.Spec, kd.Spec) {
					kd.Spec = exd.Spec
					updateDevs = append(updateDevs, kd)
				}
				break
			}
		}
		if !exist {
			exd.Namespace = ds.Namespace
			addDevs = append(addDevs, exd)
		}
	}

	return addDevs, updateDevs
}

// findDeleteDeviceProfile finds deviceprofiles that exist on the Kubernetes but not on the EdgeX
func findDeleteDeviceProfile(edgeXDevs, kubeDevs []devicev1alpha1.DeviceProfile) []devicev1alpha1.DeviceProfile {
	var deleteDevs []devicev1alpha1.DeviceProfile
	for _, kd := range kubeDevs {
		var exist bool
		for i, exd := range edgeXDevs {
			dp := edgeXDevs[i]
			if exd.Name == strings.ToLower(util.GetEdgeDeviceProfileName(&dp, EdgeXObjectName)) {
				exist = true
				break
			}
		}
		if !exist && kd.Status.Synced {
			deleteDevs = append(deleteDevs, kd)
		}
	}
	return deleteDevs
}

func getKubeNameWithPrefix(edgeName, NodePoolName string) string {
	if NodePoolName == "" {
		return edgeName
	}
	return fmt.Sprintf("%s-%s", NodePoolName, edgeName)
}

// createDeviceProfile creates the list of device profiles
func createDeviceProfile(cli client.Client, edgeXDevs []devicev1alpha1.DeviceProfile, NodePoolName string) error {
	for _, ed := range edgeXDevs {
		ed.SetName(getKubeNameWithPrefix(ed.GetName(), NodePoolName))
		if err := cli.Create(context.TODO(), &ed); err != nil {
			if apierrors.IsAlreadyExists(err) {
				klog.V(4).InfoS("DeviceProfile already exist on Kubernetes", "deviceprofile", strings.ToLower(ed.Name))
				continue
			}
			return err
		}
		if err := cli.Status().Update(context.TODO(), &ed); err != nil {
			return err
		}
		klog.V(4).InfoS("Successfully create DeviceProfile to Kubernetes", "DeviceProfile", ed.GetName())
	}
	return nil
}

func deleteDeviceProfile(cli client.Client, kubeDevs []devicev1alpha1.DeviceProfile) error {
	for _, kd := range kubeDevs {
		if err := cli.Delete(context.TODO(), &kd); err != nil {
			if apierrors.IsNotFound(err) {
				klog.V(4).InfoS("DeviceProfile doesn't exist on Kubernetes", "deviceprofile", kd.Name)
				continue
			}
			return err
		}
		klog.V(4).InfoS("Successfully delete DeviceProfile on Kubernetes", "DeviceProfile", kd.GetName())
	}
	return nil
}
