/*
Copyright 2021 The Kubernetes authors.

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

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devicev1alpha1 "github.com/charleszheng44/device-controller/api/v1alpha1"
	clis "github.com/charleszheng44/device-controller/clients"
	coremetacli "github.com/charleszheng44/device-controller/clients/core-metadata"
)

// DeviceServiceReconciler reconciles a DeviceService object
type DeviceServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	*coremetacli.CoreMetaClient
}

//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceservices/finalizers,verbs=update

func (r *DeviceServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("deviceservice", req.NamespacedName)
	var ds devicev1alpha1.DeviceService
	if err := r.Get(ctx, req.NamespacedName, &ds); err != nil {
		return ctrl.Result{}, err
	}

	_, err := r.GetDeviceServiceByName(ds.GetName())
	if err == nil {
		log.Info(
			"DeviceService already exists on EdgeX")
		return ctrl.Result{}, nil
	}
	if !clis.IsNotFoundErr(err) {
		log.Error(err, "fail to visit the EdgeX core-metatdata-service")
		return ctrl.Result{}, nil
	}

	// 1. create the addressable
	add := toEdgeXAddressable(ds.Spec.Addressable)
	_, err = r.GetAddressableByName(add.Name)
	if err == nil {
		log.Info(
			"Addressable already exists on EdgeX")
		return ctrl.Result{}, nil
	}
	addrEdgeXId, err := r.AddAddressable(add)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Fail to add addressable to EdgeX: %v", err)
	}
	log.V(4).Info("Successfully add the Addressable to EdgeX",
		"Addressable", add.Name, "EdgeXId", addrEdgeXId)
	ds.Spec.Addressable.Id = addrEdgeXId

	// 2. create the DeviceService
	dsEdgeXId, err := r.AddDeviceService(toEdgexDeviceService(ds))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Fail to add DeviceService to EdgeX: %v", err)
	}
	log.V(4).Info("Successfully add DeviceService to EdgeX",
		"DeviceService", ds.GetName(), "EdgeXId", dsEdgeXId)
	ds.Spec.Id = dsEdgeXId
	ds.Status.AddedToEdgeX = true

	return ctrl.Result{}, r.Update(ctx, &ds)
}

func toEdgexDeviceService(ds devicev1alpha1.DeviceService) models.DeviceService {
	return models.DeviceService{
		DescribedObject: models.DescribedObject{
			Description: ds.Spec.Description,
		},
		Name:           ds.GetName(),
		Id:             ds.Spec.Id,
		LastConnected:  ds.Spec.LastConnected,
		LastReported:   ds.Spec.LastReported,
		OperatingState: models.OperatingState(ds.Spec.OperatingState),
		Labels:         ds.Spec.Labels,
		AdminState:     models.AdminState(ds.Spec.AdminState),
		Addressable:    toEdgeXAddressable(ds.Spec.Addressable),
	}
}

func toEdgeXAddressable(a devicev1alpha1.Addressable) models.Addressable {
	return models.Addressable{
		Id:         a.Id,
		Name:       a.Name,
		Protocol:   a.Protocol,
		HTTPMethod: a.HTTPMethod,
		Address:    a.Address,
		Port:       a.Port,
		Path:       a.Path,
		Publisher:  a.Publisher,
		User:       a.User,
		Password:   a.Password,
		Topic:      a.Topic,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.CoreMetaClient = coremetacli.NewCoreMetaClient(
		"edgex-core-metadata.default", 48081, r.Log)
	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.DeviceService{}).
		Complete(r)
}
