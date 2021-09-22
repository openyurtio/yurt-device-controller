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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
	devcli "github.com/openyurtio/device-controller/clients"
	edgexclis "github.com/openyurtio/device-controller/clients/edgex-foundry"
	"github.com/openyurtio/device-controller/controllers/util"
)

// DeviceProfileReconciler reconciles a DeviceProfile object
type DeviceProfileReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	edgeClient devcli.DeviceProfileInterface
	NodePool   string
}

//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceprofiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceprofiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceprofiles/finalizers,verbs=update

// Reconcile make changes to a deviceprofile object in EdgeX based on it in Kubernetes
func (r *DeviceProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("deviceprofile", req.NamespacedName)
	var curdp devicev1alpha1.DeviceProfile
	if err := r.Get(ctx, req.NamespacedName, &curdp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if curdp.Spec.NodePool != r.NodePool {
		return ctrl.Result{}, nil
	}

	dpName := util.GetEdgeNameTrimNodePool(curdp.GetName(), r.NodePool)
	var prevdp devicev1alpha1.DeviceProfile
	var exist bool
	edps, err := r.edgeClient.List(context.Background(), devcli.ListOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, edp := range edps {
		if strings.ToLower(edp.Name) == dpName {
			prevdp = edp
			exist = true
			break
		}
	}

	if !curdp.ObjectMeta.DeletionTimestamp.IsZero() {
		if exist {
			if err := r.edgeClient.Delete(context.Background(), prevdp.Name, devcli.DeleteOptions{}); err != nil {
				return ctrl.Result{}, fmt.Errorf("Fail to delete DeviceProfile on Edgex: %v", err)
			}
			log.Info("Successfully delete DeviceProfile on EdgeX", "DeviceProfile", prevdp.Name)
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
		log.Info("Successfully add DeviceProfile to EdgeX",
			"DeviceProfile", curdp.GetName(), "EdgeId", curdp.Status.EdgeId)
		return ctrl.Result{}, r.Status().Update(ctx, curdp)
	}
	curdp.Spec.NodePool = ""
	if !reflect.DeepEqual(curdp.Spec, prevdp.Spec) {
		// TODO
		log.Info("controller doesn't support update deviceprofile from Kubernetes to EdgeX")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.edgeClient = edgexclis.NewEdgexDeviceProfile(
		"edgex-core-metadata", 48081, r.Log)
	nodePool, err := util.GetNodePool(mgr.GetConfig())
	if err != nil {
		return err
	}
	r.NodePool = nodePool

	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.DeviceProfile{}).
		WithEventFilter(genFirstUpdateFilter("deviceprofile", r.Log)).
		Complete(r)
}
