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
	"errors"
	"fmt"
	"net/http"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devicev1alpha1 "github.com/charleszheng44/device-controller/api/v1alpha1"
	clis "github.com/charleszheng44/device-controller/clients"
	corecmdcli "github.com/charleszheng44/device-controller/clients/core-command"
	coremetacli "github.com/charleszheng44/device-controller/clients/core-metadata"
)

// DeviceReconciler reconciles a Device object
type DeviceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	*coremetacli.CoreMetaClient
	*corecmdcli.CoreCommandClient
}

//+kubebuilder:rbac:groups=device.openyurt.io,resources=devices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=devices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=devices/finalizers,verbs=update

func (r *DeviceReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("device", req.NamespacedName)
	var d devicev1alpha1.Device
	if err := r.Get(ctx, req.NamespacedName, &d); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("Reconciling the Device object", "Device", d.GetName(), "AddedToEdgeX", d.Status.AddedToEdgeX)

	if d.Status.AddedToEdgeX == true {
		// the device has been added to the EdgeX foundry,
		// check if each device property are in the desired state
		for _, dps := range d.Spec.DeviceProperties {
			log.Info("getting the actual property state", "property", dps.Name)
			aps, err := getActualPropertyState(dps.Name, &d, r.CoreCommandClient)
			if err != nil {
				return ctrl.Result{}, err
			}
			log.Info("got the actual property state",
				"property name", aps.Name,
				"property getURL", aps.GetURL,
				"property actual value", aps.ActualValue)
			if d.Status.DeviceProperties == nil {
				d.Status.DeviceProperties = map[string]devicev1alpha1.ActualPropertyState{}
			}
			d.Status.DeviceProperties[aps.Name] = aps
			if dps.DesiredValue != aps.ActualValue {
				log.Info("the desired value and the actual value are different",
					"desired value", dps.DesiredValue,
					"actual value", aps.ActualValue)
				if dps.PutURL == "" {
					putURL, err := getPutURL(d.GetName(), dps.Name, r.CoreCommandClient)
					if err != nil {
						return ctrl.Result{}, err
					}
					dps.PutURL = putURL
					log.Info("get the desired property putURL",
						"property", dps.Name, "putURL", putURL)
				}
				// set the device property to desired state
				log.Info("setting the property to desired value", "property", dps.Name)
				rep, err := resty.New().R().
					SetHeader("Content-Type", "application/json").
					SetBody([]byte(fmt.Sprintf(`{"%s": "%s"}`, dps.Name, dps.DesiredValue))).
					Put(dps.PutURL)
				if err != nil {
					return ctrl.Result{}, err
				}
				if rep.StatusCode() == http.StatusOK {
					log.Info("successfully set the property to desired value", "property", dps.Name)
					log.Info("setting the actual property value to desired value", "property", dps.Name)
					// if the device property has been successfully set, we will
					// update the Device.Status.DeviceProperties[name] as well
					if d.Status.DeviceProperties == nil {
						d.Status.DeviceProperties = map[string]devicev1alpha1.ActualPropertyState{}
					}
					oldAps, exist := d.Status.DeviceProperties[dps.Name]
					if !exist {
						d.Status.DeviceProperties[dps.Name] = devicev1alpha1.ActualPropertyState{
							Name:        dps.Name,
							ActualValue: dps.DesiredValue,
						}
						continue
					}
					oldAps.ActualValue = dps.DesiredValue
					d.Status.DeviceProperties[dps.Name] = oldAps
					log.Info("set the actual property value to desired value", "property", dps.Name)
				}
			}
		}
		return ctrl.Result{}, r.Status().Update(ctx, &d)
	}

	log.Info("Checking if device already exist on the EdgeX", "device", d.GetName())
	_, err := r.GetDeviceByName(d.GetName())
	if err == nil {
		log.Info("Device already exists on EdgeX")
		return ctrl.Result{}, nil
	}
	if !clis.IsNotFoundErr(err) {
		log.Info("fail to visit the EdgeX core-metadata-service")
		return ctrl.Result{}, nil
	}

	log.Info("Adding device to the EdgeX", "device", d.GetName())
	edgeXId, err := r.AddDevice(toEdgeXDevice(d))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Fail to add Device to EdgeX: %v", err)
	}
	log.Info("Successfully add Device to EdgeX",
		"Device", d.GetName(), "EdgeXId", edgeXId)
	d.Status.Id = edgeXId
	d.Status.AddedToEdgeX = true
	return ctrl.Result{Requeue: true}, r.Status().Update(ctx, &d)
}

func getPutURL(name, cmdName string, cli *corecmdcli.CoreCommandClient) (string, error) {
	cr, err := cli.GetCommandResponseByName(name)
	if err != nil {
		return "", err
	}
	for _, c := range cr.Commands {
		if cmdName == c.Name {
			return c.Put.URL, nil
		}
	}
	return "", errors.New("corresponding command is not found")
}

func getActualPropertyState(
	name string,
	d *devicev1alpha1.Device,
	cli *corecmdcli.CoreCommandClient) (devicev1alpha1.ActualPropertyState, error) {
	oldAps, exist := d.Status.DeviceProperties[name]
	if !exist {
		aps := devicev1alpha1.ActualPropertyState{}
		aps.Name = name
		// get the command path
		cr, err := cli.GetCommandResponseByName(d.GetName())
		if err != nil {
			return devicev1alpha1.ActualPropertyState{}, err
		}
		for _, c := range cr.Commands {
			if name == c.Name {
				aps.GetURL = c.Get.URL
			}
		}
		// get the actual state from the EdgeX
		resp, err := resty.New().R().Get(aps.GetURL)
		if err != nil {
			return devicev1alpha1.ActualPropertyState{}, err
		}
		// TODO check the response message
		aps.ActualValue = string(resp.Body())
		if d.Status.DeviceProperties == nil {
			d.Status.DeviceProperties = map[string]devicev1alpha1.ActualPropertyState{}
		}
		d.Status.DeviceProperties[name] = aps
		return aps, err
	}
	if oldAps.GetURL == "" {
		// get the command path
		cr, err := cli.GetCommandResponseByName(name)
		if err != nil {
			return devicev1alpha1.ActualPropertyState{}, err
		}
		for _, c := range cr.Commands {
			if name == c.Name {
				oldAps.GetURL = c.Get.URL
			}
		}
		// get the actual state from the EdgeX
		resp, err := resty.New().R().Get(oldAps.GetURL)
		if err != nil {
			return devicev1alpha1.ActualPropertyState{}, err
		}
		// TODO check the response message
		oldAps.ActualValue = string(resp.Body())
		if d.Status.DeviceProperties == nil {
			d.Status.DeviceProperties = map[string]devicev1alpha1.ActualPropertyState{}
		}
		d.Status.DeviceProperties[name] = oldAps
		return oldAps, err
	}
	// get the actual state from the EdgeX
	resp, err := resty.New().R().Get(oldAps.GetURL)
	if err != nil {
		return devicev1alpha1.ActualPropertyState{}, err
	}
	// TODO check the response message
	oldAps.ActualValue = string(resp.Body())
	if d.Status.DeviceProperties == nil {
		d.Status.DeviceProperties = map[string]devicev1alpha1.ActualPropertyState{}
	}
	d.Status.DeviceProperties[name] = oldAps
	return oldAps, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.CoreMetaClient = coremetacli.NewCoreMetaClient(
		"edgex-core-metadata.default", 48081, r.Log)
	r.CoreCommandClient = corecmdcli.NewCoreCommandClient(
		"edgex-core-command.default", 48082, r.Log)
	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.Device{}).
		WithEventFilter(genFirstUpdateFilter("device", r.Log)).
		Complete(r)
}

func toEdgeXDevice(d devicev1alpha1.Device) models.Device {
	md := models.Device{
		DescribedObject: models.DescribedObject{
			Description: d.Spec.Description,
		},
		Id:             d.Status.Id,
		Name:           d.GetName(),
		AdminState:     toEdgeXAdminState(d.Spec.AdminState),
		OperatingState: toEdgeXOperatingState(d.Spec.OperatingState),
		Protocols:      toEdgeXProtocols(d.Spec.Protocols),
		LastConnected:  d.Status.LastConnected,
		LastReported:   d.Status.LastReported,
		Labels:         d.Spec.Labels,
		Location:       d.Spec.Location,
		Service:        models.DeviceService{Name: d.Spec.Service},
		Profile: toEdgeXDeviceProfile(
			devicev1alpha1.DeviceProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name: d.Spec.Profile,
				},
			},
		),
	}
	if d.Status.Id != "" {
		md.Id = d.Status.Id
	}
	return md
}

func toEdgeXProtocols(
	pps map[string]devicev1alpha1.ProtocolProperties) map[string]models.ProtocolProperties {
	ret := map[string]models.ProtocolProperties{}
	for k, v := range pps {
		ret[k] = models.ProtocolProperties(v)
	}
	return ret
}

func toEdgeXAdminState(as devicev1alpha1.AdminState) models.AdminState {
	if as == devicev1alpha1.Locked {
		return models.Locked
	}
	return models.Unlocked
}

func toEdgeXOperatingState(os devicev1alpha1.OperatingState) models.OperatingState {
	if os == devicev1alpha1.Enabled {
		return models.Enabled
	}
	return models.Disabled
}
