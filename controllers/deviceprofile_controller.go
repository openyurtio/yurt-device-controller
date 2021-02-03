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

// DeviceProfileReconciler reconciles a DeviceProfile object
type DeviceProfileReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	*coremetacli.CoreMetaClient
}

//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceprofiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceprofiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=deviceprofiles/finalizers,verbs=update

func (r *DeviceProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("deviceprofile", req.NamespacedName)
	var dp devicev1alpha1.DeviceProfile
	if err := r.Get(ctx, req.NamespacedName, &dp); err != nil {
		return ctrl.Result{}, err
	}

	_, err := r.GetDeviceProfileByName(dp.GetName())
	if err == nil {
		log.Info(
			"DeviceProfile already exists on EdgeX")
		return ctrl.Result{}, nil
	}

	if !clis.IsNotFoundErr(err) {
		log.Info("Fail to visit the EdgeX core-metadata-service")
		return ctrl.Result{}, nil
	}

	edgeXId, err := r.AddDeviceProfile(toEdgeXDeviceProfile(dp))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Fail to add DeviceProfile to Edgex: %v", err)
	}
	log.V(4).Info("Successfully add DeviceProfile to EdgeX",
		"DeviceProfile", dp.GetName(), "EdgeXId", edgeXId)
	dp.Spec.Id = edgeXId
	dp.Status.AddedToEdgeX = true
	return ctrl.Result{}, r.Update(ctx, &dp)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.CoreMetaClient = coremetacli.NewCoreMetaClient(
		"edgex-core-metadata.default", 48081, r.Log)
	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.DeviceProfile{}).
		WithEventFilter(genFirstUpdateFilter("deviceprofile", r.Log)).
		Complete(r)
}

func toEdgeXDeviceProfile(
	dp devicev1alpha1.DeviceProfile) models.DeviceProfile {
	cs := []models.Command{}
	for _, c := range dp.Spec.CoreCommands {
		cs = append(cs, toEdgeXCommand(c))
	}
	dcs := []models.ProfileResource{}
	for _, pr := range dp.Spec.DeviceCommands {
		dcs = append(dcs, toEdgeXProfileResource(pr))
	}

	return models.DeviceProfile{
		DescribedObject: models.DescribedObject{
			Description: dp.Spec.Description,
		},
		Id:              dp.Spec.Id,
		Name:            dp.GetName(),
		Manufacturer:    dp.Spec.Manufacturer,
		Model:           dp.Spec.Model,
		Labels:          dp.Spec.Labels,
		DeviceResources: toEdgeXDeviceResourceSlice(dp.Spec.DeviceResources),
		DeviceCommands:  dcs,
		CoreCommands:    cs,
	}
}

func toEdgeXDeviceResourceSlice(
	drs []devicev1alpha1.DeviceResource) []models.DeviceResource {
	var ret []models.DeviceResource
	for _, dr := range drs {
		ret = append(ret, toEdgeXDeviceResource(dr))
	}
	return ret
}

func toEdgeXDeviceResource(
	dr devicev1alpha1.DeviceResource) models.DeviceResource {
	return models.DeviceResource{
		Description: dr.Description,
		Name:        dr.Name,
		Tag:         dr.Tag,
		Properties:  toEdgeXProfileProperty(dr.Properties),
		Attributes:  dr.Attributes,
	}
}

func toEdgeXProfileProperty(
	pp devicev1alpha1.ProfileProperty) models.ProfileProperty {
	return models.ProfileProperty{
		Value: toEdgeXPropertyValue(pp.Value),
		Units: toEdgeXUnits(pp.Units),
	}
}

func toEdgeXPropertyValue(
	pv devicev1alpha1.PropertyValue) models.PropertyValue {
	return models.PropertyValue{
		Type:          pv.Type,
		ReadWrite:     pv.ReadWrite,
		Minimum:       pv.Minimum,
		Maximum:       pv.Maximum,
		DefaultValue:  pv.DefaultValue,
		Size:          pv.Size,
		Mask:          pv.Mask,
		Shift:         pv.Shift,
		Scale:         pv.Scale,
		Offset:        pv.Offset,
		Base:          pv.Base,
		Assertion:     pv.Assertion,
		Precision:     pv.Precision,
		FloatEncoding: pv.FloatEncoding,
		MediaType:     pv.MediaType,
	}
}

func toEdgeXUnits(u devicev1alpha1.Units) models.Units {
	return models.Units{
		Type:         u.Type,
		ReadWrite:    u.ReadWrite,
		DefaultValue: u.DefaultValue,
	}
}

func toEdgeXCommand(c devicev1alpha1.Command) models.Command {
	return models.Command{
		Id:   c.Id,
		Name: c.Name,
		Get:  toEdgeXGet(c.Get),
		Put:  toEdgeXPut(c.Put),
	}
}

func toEdgeXPut(p devicev1alpha1.Put) models.Put {
	return models.Put{
		Action:         toEdgeXAction(p.Action),
		ParameterNames: p.ParameterNames,
	}
}

func toEdgeXGet(g devicev1alpha1.Get) models.Get {
	return models.Get{
		Action: toEdgeXAction(g.Action),
	}
}

func toEdgeXAction(a devicev1alpha1.Action) models.Action {
	responses := []models.Response{}
	for _, r := range a.Responses {
		responses = append(responses, toEdgeXResponse(r))
	}
	return models.Action{
		Path:      a.Path,
		Responses: responses,
		URL:       a.URL,
	}
}

func toEdgeXResponse(r devicev1alpha1.Response) models.Response {
	return models.Response{
		Code:           r.Code,
		Description:    r.Description,
		ExpectedValues: r.ExpectedValues,
	}
}

func toEdgeXProfileResource(pr devicev1alpha1.ProfileResource) models.ProfileResource {
	gros := []models.ResourceOperation{}
	for _, gro := range pr.Get {
		gros = append(gros, toEdgeXResourceOperation(gro))
	}
	sros := []models.ResourceOperation{}
	for _, sro := range pr.Set {
		sros = append(sros, toEdgeXResourceOperation(sro))
	}
	return models.ProfileResource{
		Name: pr.Name,
		Get:  gros,
		Set:  sros,
	}
}

func toEdgeXResourceOperation(ro devicev1alpha1.ResourceOperation) models.ResourceOperation {
	return models.ResourceOperation{
		Index:          ro.Index,
		Operation:      ro.Operation,
		Object:         ro.Object,
		DeviceResource: ro.DeviceResource,
		Parameter:      ro.Parameter,
		Resource:       ro.Resource,
		DeviceCommand:  ro.DeviceCommand,
		Secondary:      ro.Secondary,
		Mappings:       ro.Mappings,
	}
}
