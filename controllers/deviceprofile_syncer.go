package controllers

import (
	"context"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	coremetacli "github.com/charleszheng44/device-controller/clients/core-metadata"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devv1 "github.com/charleszheng44/device-controller/api/v1alpha1"
)

type DeviceProfileSyncer struct {
	// syncing period in seconds
	syncPeriod time.Duration
	// EdgeX core-data-service's client
	*coremetacli.CoreMetaClient
	// Kubernetes client
	client.Client
	log logr.Logger
}

// NewDeviceProfileSyncer initialize a New DeviceProfileSyncer
func NewDeviceProfileSyncer(client client.Client,
	logr logr.Logger,
	periodSecs uint32) DeviceProfileSyncer {
	log := logr.WithName("syncer").WithName("DeviceProfile")
	return DeviceProfileSyncer{
		syncPeriod: time.Duration(periodSecs) * time.Second,
		CoreMetaClient: coremetacli.NewCoreMetaClient(
			"edgex-core-metadata.default", 48081, log),
		Client: client,
		log:    log,
	}
}

// NewDeviceProfileSyncerRunnablel initialize a controller-runtime manager runnable
func (dps *DeviceProfileSyncer) NewDeviceProfileSyncerRunnable() ctrlmgr.RunnableFunc {
	return func(ctx context.Context) error {
		dps.Run(ctx.Done())
		return nil
	}
}

func (ds *DeviceProfileSyncer) Run(stop <-chan struct{}) {
	ds.log.Info("starting the DeviceProfileSyncer...")
	go func() {
		for {
			<-time.After(ds.syncPeriod)
			// list devices on edgex foundry
			eDevs, err := ds.ListDeviceProfile()
			if err != nil {
				ds.log.Error(err, "fail to list the deviceprofile object on the EdgeX Foundry")
				continue
			}
			// list devices on Kubernetes
			var kDevs devv1.DeviceProfileList
			if err := ds.List(context.TODO(), &kDevs); err != nil {
				ds.log.Error(err, "fail to list the deviceprofile object on the Kubernetes")
				continue
			}
			// create the devices on Kubernetes but not on EdgeX
			newKDevs := findNewDeviceProfile(eDevs, kDevs.Items)
			if len(newKDevs) != 0 {
				if err := createDeviceProfile(ds.log, ds.Client, newKDevs); err != nil {
					ds.log.Error(err, "fail to create devices profile")
					continue
				}
			}
			ds.log.V(5).Info("new deviceprofile not found")
		}
	}()

	<-stop
	ds.log.Info("stopping the deviceprofile syncer")
}

// findNewDeviceProfile finds deviceprofiles that have been created on the EdgeX but
// not the Kubernetes
func findNewDeviceProfile(
	edgeXDevs []models.DeviceProfile,
	kubeDevs []devv1.DeviceProfile) []models.DeviceProfile {
	var retDevs []models.DeviceProfile
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

// createDeviceProfile creates the list of devices
func createDeviceProfile(log logr.Logger, cli client.Client, edgeXDevs []models.DeviceProfile) error {
	for _, ed := range edgeXDevs {
		kd := toKubeDeviceProfile(ed)
		if err := cli.Create(context.TODO(), &kd); err != nil {
			if apierrors.IsAlreadyExists(err) {
				log.Info("DeviceProfile already exist on Kubernetes",
					"deviceprofile", strings.ToLower(ed.Name))
				continue
			}
			log.Error(err, "fail to create the DeviceProfile on Kubernetes",
				"deviceprofile", ed.Name)
			return err
		}
	}
	return nil
}

func toKubeDeviceProfile(dp models.DeviceProfile) devv1.DeviceProfile {
	return devv1.DeviceProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(dp.Name),
			Namespace: "default",
			Labels: map[string]string{
				EdgeXObjectName: dp.Name,
			},
		},
		Spec: devv1.DeviceProfileSpec{
			Description:     dp.Description,
			Id:              dp.Id,
			Manufacturer:    dp.Manufacturer,
			Model:           dp.Model,
			Labels:          dp.Labels,
			DeviceResources: toKubeDeviceResources(dp.DeviceResources),
			CoreCommands:    toKubeCoreCommands(dp.CoreCommands),
		},
		Status: devv1.DeviceProfileStatus{
			AddedToEdgeX: true,
		},
	}
}

func toKubeDeviceResources(drs []models.DeviceResource) []devv1.DeviceResource {
	var ret []devv1.DeviceResource
	for _, dr := range drs {
		ret = append(ret, toKubeDeviceResource(dr))
	}
	return ret
}

func toKubeDeviceResource(dr models.DeviceResource) devv1.DeviceResource {
	return devv1.DeviceResource{
		Description: dr.Description,
		Name:        dr.Name,
		Tag:         dr.Tag,
		Properties:  toKubeProfileProperty(dr.Properties),
		Attributes:  dr.Attributes,
	}
}

func toKubeProfileProperty(pp models.ProfileProperty) devv1.ProfileProperty {
	return devv1.ProfileProperty{
		Value: toKubePropertyValue(pp.Value),
		Units: toKubeUnits(pp.Units),
	}
}

func toKubePropertyValue(pv models.PropertyValue) devv1.PropertyValue {
	return devv1.PropertyValue{
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

func toKubeUnits(u models.Units) devv1.Units {
	return devv1.Units{
		Type:         u.Type,
		ReadWrite:    u.ReadWrite,
		DefaultValue: u.DefaultValue,
	}
}

func toKubeCoreCommands(ccs []models.Command) []devv1.Command {
	var ret []devv1.Command
	for _, cc := range ccs {
		ret = append(ret, toKubeCoreCommand(cc))
	}
	return ret
}

func toKubeCoreCommand(cc models.Command) devv1.Command {
	return devv1.Command{
		Name: cc.Name,
		Id:   cc.Id,
		Get:  toKubeGet(cc.Get),
		Put:  toKubePut(cc.Put),
	}
}

func toKubeGet(get models.Get) devv1.Get {
	return devv1.Get{
		Action: toKubeAction(get.Action),
	}
}

func toKubePut(put models.Put) devv1.Put {
	return devv1.Put{
		Action:         toKubeAction(put.Action),
		ParameterNames: put.ParameterNames,
	}
}

func toKubeAction(act models.Action) devv1.Action {
	return devv1.Action{
		Path:      act.Path,
		Responses: toKubeResponses(act.Responses),
		URL:       act.URL,
	}
}

func toKubeResponses(reps []models.Response) []devv1.Response {
	var ret []devv1.Response
	for _, rep := range reps {
		ret = append(ret, toKubeResponse(rep))
	}
	return ret
}

func toKubeResponse(rep models.Response) devv1.Response {
	return devv1.Response{
		Code:           rep.Code,
		Description:    rep.Description,
		ExpectedValues: rep.ExpectedValues,
	}
}
