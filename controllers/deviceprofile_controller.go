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
	var curdp devicev1alpha1.DeviceProfile
	if err := r.Get(ctx, req.NamespacedName, &curdp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if curdp.Spec.NodePool != r.NodePool {
		return ctrl.Result{}, nil
	}

	klog.V(3).Infof("Reconciling the DeviceProfile: %s", curdp.GetName())
	dpActualName := util.GetEdgeDeviceProfileName(&curdp, EdgeXObjectName)
	var prevdp *devicev1alpha1.DeviceProfile
	var exist bool
	prevdp, err := r.edgeClient.Get(context.Background(), dpActualName, devcli.GetOptions{})
	if err == nil {
		exist = true
	} else if clis.IsNotFoundErr(err) {
		exist = false
	} else {
		return ctrl.Result{}, err
	}

	if !curdp.ObjectMeta.DeletionTimestamp.IsZero() {
		if exist {
			if err := r.edgeClient.Delete(context.Background(), dpActualName, devcli.DeleteOptions{}); err != nil {
				return ctrl.Result{}, fmt.Errorf("Fail to delete DeviceProfile on Edgex: %v", err)
			}
			klog.V(2).Infof("Successfully delete DeviceProfile on edge platform: %s", dpActualName)
		}
		controllerutil.RemoveFinalizer(&curdp, "devicecontroller.openyurt.io")
		err := r.Update(context.TODO(), &curdp)

		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(&curdp, "devicecontroller.openyurt.io") {
		controllerutil.AddFinalizer(&curdp, "devicecontroller.openyurt.io")
		if err = r.Update(context.TODO(), &curdp); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	}
	if !exist {
		curdp, err := r.edgeClient.Create(context.Background(), &curdp, devcli.CreateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("Fail to add DeviceProfile to Edgex: %v", err)
		}
		klog.V(2).Infof("Successfully add DeviceProfile to edge platform, Name: %s, EdgeId: %s", curdp.GetName(), curdp.Status.EdgeId)
		return ctrl.Result{}, r.Status().Update(ctx, curdp)
	}
	curdp.Spec.NodePool = ""
	if !reflect.DeepEqual(curdp.Spec, prevdp.Spec) {
		// TODO
		klog.V(2).Info("controller doesn't support update deviceprofile from Kubernetes to edge platform")
		return ctrl.Result{}, nil
	}
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
