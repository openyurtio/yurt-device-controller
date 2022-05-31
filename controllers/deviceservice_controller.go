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
	"encoding/json"
	"fmt"

	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
	clis "github.com/openyurtio/device-controller/clients"
	edgeInterface "github.com/openyurtio/device-controller/clients"
	edgexCli "github.com/openyurtio/device-controller/clients/edgex-foundry"
	"github.com/openyurtio/device-controller/cmd/yurt-device-controller/options"
	"github.com/openyurtio/device-controller/controllers/util"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeviceServiceReconciler reconciles a DeviceService object
type DeviceServiceReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	deviceServiceCli edgeInterface.DeviceServiceInterface
	NodePool         string
}

//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceservices/finalizers,verbs=update

func (r *DeviceServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ds devicev1alpha1.DeviceService
	if err := r.Get(ctx, req.NamespacedName, &ds); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If objects doesn't belong to the edge platform to which the controller is connected, the controller does not handle events for that object
	if ds.Spec.NodePool != r.NodePool {
		return ctrl.Result{}, nil
	}
	klog.V(3).Infof("Reconciling the DeviceService: %s", ds.GetName())
	// Update deviceService conditions
	defer func() {
		if ds.Spec.Managed != true {
			conditions.MarkFalse(&ds, devicev1alpha1.DeviceServiceManagingCondition, "this deviceService is not managed by openyurt", clusterv1.ConditionSeverityInfo, "")
		}
		conditions.SetSummary(&ds,
			conditions.WithConditions(
				devicev1alpha1.DeviceServiceSyncedCondition, devicev1alpha1.DeviceServiceManagingCondition),
		)
		err := r.Status().Update(ctx, &ds)
		if client.IgnoreNotFound(err) != nil {
			if !apierrors.IsConflict(err) {
				klog.V(4).ErrorS(err, "update deviceService conditions failed", "deviceService", ds.GetName())
			}
		}
	}()

	// 1. Handle the deviceService deletion event
	if err := r.reconcileDeleteDeviceService(ctx, &ds); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else if !ds.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if ds.Status.Synced == false {
		// 2. Synchronize OpenYurt deviceService to edge platform
		if err := r.reconcileCreateDeviceService(ctx, &ds); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				return ctrl.Result{}, err
			}
		}
	} else if ds.Spec.Managed == true {
		// 3. If the deviceService has been synchronized and is managed by the cloud, reconcile the deviceService fields
		if err := r.reconcileUpdateDeviceService(ctx, &ds); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceServiceReconciler) SetupWithManager(mgr ctrl.Manager, opts *options.YurtDeviceControllerOptions) error {
	r.deviceServiceCli = edgexCli.NewEdgexDeviceServiceClient(opts.CoreMetadataAddr)
	r.NodePool = opts.Nodepool

	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.DeviceService{}).
		Complete(r)
}

func (r *DeviceServiceReconciler) reconcileDeleteDeviceService(ctx context.Context, ds *devicev1alpha1.DeviceService) error {
	// gets the actual name of deviceService on the edge platform from the Label of the device
	edgeDeviceServiceName := util.GetEdgeDeviceServiceName(ds, EdgeXObjectName)
	if ds.ObjectMeta.DeletionTimestamp.IsZero() {
		if len(ds.GetFinalizers()) == 0 {
			patchString := map[string]interface{}{
				"metadata": map[string]interface{}{
					"finalizers": []string{devicev1alpha1.DeviceServiceFinalizer},
				},
			}
			if patchData, err := json.Marshal(patchString); err != nil {
				return err
			} else {
				if err = r.Patch(ctx, ds, client.RawPatch(types.MergePatchType, patchData)); err != nil {
					return err
				}
			}
		}
	} else {
		patchString := map[string]interface{}{
			"metadata": map[string]interface{}{
				"finalizers": []string{},
			},
		}
		// delete the deviceService in OpenYurt
		if patchData, err := json.Marshal(patchString); err != nil {
			return err
		} else {
			if err = r.Patch(ctx, ds, client.RawPatch(types.MergePatchType, patchData)); err != nil {
				return err
			}
		}

		// delete the deviceService object on edge platform
		err := r.deviceServiceCli.Delete(nil, edgeDeviceServiceName, edgeInterface.DeleteOptions{})
		if err != nil && !clis.IsNotFoundErr(err) {
			return err
		}
	}
	return nil
}

func (r *DeviceServiceReconciler) reconcileCreateDeviceService(ctx context.Context, ds *devicev1alpha1.DeviceService) error {
	// get the actual name of deviceService on the Edge platform from the Label of the device
	edgeDeviceServiceName := util.GetEdgeDeviceServiceName(ds, EdgeXObjectName)
	klog.V(4).Infof("Checking if deviceService already exist on the edge platform: %s", ds.GetName())
	// Checking if deviceService already exist on the edge platform
	if edgeDs, err := r.deviceServiceCli.Get(nil, edgeDeviceServiceName, edgeInterface.GetOptions{}); err != nil {
		if !clis.IsNotFoundErr(err) {
			klog.V(4).ErrorS(err, "fail to visit the edge platform")
			return nil
		} else {
			createdDs, err := r.deviceServiceCli.Create(nil, ds, edgeInterface.CreateOptions{})
			if err != nil {
				klog.V(4).ErrorS(err, "failed to create deviceService on edge platform")
				conditions.MarkFalse(ds, devicev1alpha1.DeviceServiceSyncedCondition, "failed to add DeviceService to EdgeX", clusterv1.ConditionSeverityWarning, err.Error())
				return fmt.Errorf("fail to add DeviceService to edge platform: %v", err)
			}

			klog.V(4).Infof("Successfully add DeviceService to Edge Platform, Name: %s, EdgeId: %s", ds.GetName(), createdDs.Status.EdgeId)
			ds.Status.EdgeId = createdDs.Status.EdgeId
			ds.Status.Synced = true
			conditions.MarkTrue(ds, devicev1alpha1.DeviceServiceSyncedCondition)
			return r.Status().Update(ctx, ds)
		}
	} else {
		// a. If object exists, the status of the device on OpenYurt is updated
		klog.V(4).Infof("DeviceServiceName: %s, obj already exists on edge platform", ds.GetName())
		ds.Status.Synced = true
		ds.Status.EdgeId = edgeDs.Status.EdgeId
		return r.Status().Update(ctx, ds)
	}
}

func (r *DeviceServiceReconciler) reconcileUpdateDeviceService(ctx context.Context, ds *devicev1alpha1.DeviceService) error {
	// 1. reconciling the AdminState field of deviceService
	newDeviceServiceStatus := ds.Status.DeepCopy()
	updateDeviceService := ds.DeepCopy()

	if ds.Spec.AdminState != "" && ds.Spec.AdminState != ds.Status.AdminState {
		newDeviceServiceStatus.AdminState = ds.Spec.AdminState
	} else {
		updateDeviceService.Spec.AdminState = ""
	}

	_, err := r.deviceServiceCli.Update(nil, updateDeviceService, edgeInterface.UpdateOptions{})
	if err != nil {
		conditions.MarkFalse(ds, devicev1alpha1.DeviceServiceManagingCondition, "failed to update AdminState of deviceService on edge platform", clusterv1.ConditionSeverityWarning, err.Error())
		return err
	}

	// 2. update the device status on OpenYurt
	ds.Status = *newDeviceServiceStatus
	if err = r.Status().Update(ctx, ds); err != nil {
		conditions.MarkFalse(ds, devicev1alpha1.DeviceServiceManagingCondition, "failed to update status of deviceService on openyurt", clusterv1.ConditionSeverityWarning, err.Error())
		return err
	}
	conditions.MarkTrue(ds, devicev1alpha1.DeviceServiceManagingCondition)
	return nil
}
