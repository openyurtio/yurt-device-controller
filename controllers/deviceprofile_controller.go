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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
	clis "github.com/openyurtio/device-controller/clients"
	devcli "github.com/openyurtio/device-controller/clients"
	edgexclis "github.com/openyurtio/device-controller/clients/edgex-foundry"
	"github.com/openyurtio/device-controller/cmd/yurt-device-controller/options"
	"github.com/openyurtio/device-controller/controllers/util"
)

// DeviceProfileReconciler reconciles a DeviceProfile object
type DeviceProfileReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	edgeClient devcli.DeviceProfileInterface
	NodePool   string
}

//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceprofiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceprofiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceprofiles/finalizers,verbs=update

// Reconcile make changes to a deviceprofile object in EdgeX based on it in Kubernetes
func (r *DeviceProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var dp *devicev1alpha1.DeviceProfile
	if err := r.Get(ctx, req.NamespacedName, dp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if dp.Spec.NodePool != r.NodePool {
		return ctrl.Result{}, nil
	}
	klog.V(3).Infof("Reconciling the DeviceProfile: %s", dp.GetName())

	// gets the actual name of deviceProfile on the edge platform from the Label of the deviceProfile
	dpActualName := util.GetEdgeDeviceProfileName(dp, EdgeXObjectName)

	// 1. Handle the deviceProfile deletion event
	if dp.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(dp, devicev1alpha1.DeviceProfileFinalizer) {
			controllerutil.AddFinalizer(dp, devicev1alpha1.DeviceProfileFinalizer)
			if err := r.Update(ctx, dp); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(dp, devicev1alpha1.DeviceProfileFinalizer) {
			// delete the deviceProfile object on edge platform
			if err := r.edgeClient.Delete(ctx, dpActualName, devcli.DeleteOptions{}); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(dp, devicev1alpha1.DeviceProfileFinalizer)
			if err := r.Update(ctx, dp); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if dp.Status.Synced == false {
		// 2. Synchronize OpenYurt deviceProfile to edge platform
		if err := r.reconcileCreateDeviceProfile(ctx, dp, dpActualName); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				return ctrl.Result{}, err
			}
		}
	}
	// 3. Handle the deviceProfile update event
	// TODO

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceProfileReconciler) SetupWithManager(mgr ctrl.Manager, opts *options.YurtDeviceControllerOptions) error {
	r.edgeClient = edgexclis.NewEdgexDeviceProfile(opts.CoreMetadataAddr)
	r.NodePool = opts.Nodepool

	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.DeviceProfile{}).
		WithEventFilter(genFirstUpdateFilter("deviceprofile")).
		Complete(r)
}

func (r *DeviceProfileReconciler) reconcileCreateDeviceProfile(ctx context.Context, dp *devicev1alpha1.DeviceProfile, actualName string) error {
	klog.V(4).Infof("Checking if deviceProfile already exist on the edge platform: %s", dp.GetName())
	if edgeDp, err := r.edgeClient.Get(nil, actualName, devcli.GetOptions{}); err != nil {
		if !clis.IsNotFoundErr(err) {
			klog.V(4).ErrorS(err, "fail to visit the edge platform")
			return nil
		}
	} else {
		// a. If object exists, the status of the deviceProfile on OpenYurt is updated
		klog.V(4).Info("DeviceProfile already exists on edge platform")
		dp.Status.Synced = true
		dp.Status.EdgeId = edgeDp.Status.EdgeId
		return r.Status().Update(ctx, dp)
	}

	// b. If object does not exist, a request is sent to the edge platform to create a new deviceProfile
	createDp, err := r.edgeClient.Create(context.Background(), dp, devcli.CreateOptions{})
	if err != nil {
		klog.V(4).ErrorS(err, "failed to create deviceProfile on edge platform")
		return fmt.Errorf("failed to add deviceProfile to edge platform: %v", err)
	}
	klog.V(3).Infof("Successfully add DeviceProfile to edge platform, Name: %s, EdgeId: %s", createDp.GetName(), createDp.Status.EdgeId)
	dp.Status.EdgeId = createDp.Status.EdgeId
	dp.Status.Synced = true
	return r.Status().Update(ctx, dp)
}
