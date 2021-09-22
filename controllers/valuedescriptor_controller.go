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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
	clis "github.com/openyurtio/device-controller/clients"
	coredatacli "github.com/openyurtio/device-controller/clients/core-data"
)

// ValueDescriptorReconciler reconciles a ValueDescriptor object
type ValueDescriptorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	*coredatacli.CoreDataClient
}

//+kubebuilder:rbac:groups=device.openyurt.io,resources=valuedescriptors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=valuedescriptors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=valuedescriptors/finalizers,verbs=update

func (r *ValueDescriptorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("valuedescriptor", req.NamespacedName)
	var vd devicev1alpha1.ValueDescriptor
	if err := r.Get(ctx, req.NamespacedName, &vd); err != nil {
		return ctrl.Result{}, err
	}
	// 1. check if the Edgex code-data has the corresponding ValueDescriptor
	// NOTE this version does not support valuedescriptor update
	_, err := r.GetValueDescriptorByName(vd.GetName())
	if err == nil {
		log.Info("ValueDescriptor already exists on EdgeX")
		return ctrl.Result{}, nil
	}
	if !clis.IsNotFoundErr(err) {
		log.Info("Fail to visit the Edgex core-data-service")
		return ctrl.Result{}, nil
	}

	// 2. create one if the ValueDescriptor doesnot exist
	edgexId, err := r.AddValueDescript(toEdgexValue(vd))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Fail to add ValueDescriptor to Edgex: %v", err)
	}
	log.V(4).Info("Successfully add ValueDescriptor to Edgex",
		"ValueDescriptor", vd.GetName(), "EdgexId", edgexId)
	vd.Spec.Id = edgexId
	vd.Status.AddedToEdgeX = true

	return ctrl.Result{}, r.Update(ctx, &vd)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ValueDescriptorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.CoreDataClient = coredatacli.NewCoreDataClient("edgex-core-data", 48080, r.Log)
	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.ValueDescriptor{}).
		WithEventFilter(genFirstUpdateFilter("valuedescriptor", r.Log)).
		Complete(r)
}

func toEdgexValue(vd devicev1alpha1.ValueDescriptor) models.ValueDescriptor {
	return models.ValueDescriptor{
		Id:            vd.Spec.Id,
		Created:       vd.Spec.Created,
		Description:   vd.Spec.Description,
		Modified:      vd.Spec.Modified,
		Origin:        vd.Spec.Origin,
		Name:          vd.GetName(),
		Min:           vd.Spec.Min,
		Max:           vd.Spec.Max,
		DefaultValue:  vd.Spec.DefaultValue,
		Type:          vd.Spec.Type,
		UomLabel:      vd.Spec.UomLabel,
		Formatting:    vd.Spec.Formatting,
		Labels:        vd.Spec.Labels,
		MediaType:     vd.Spec.MediaType,
		FloatEncoding: vd.Spec.FloatEncoding,
	}
}
