/*
Copyright 2022 The OpenYurt Authors.

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

package options

import (
	"fmt"
	"net"

	"github.com/spf13/pflag"
)

// YurtDeviceControllerOptions is the main settings for the yurt-device-controller
type YurtDeviceControllerOptions struct {
	MetricsAddr          string
	ProbeAddr            string
	EnableLeaderElection bool
	Nodepool             string
	Namespace            string
	CoreDataAddr         string
	CoreMetadataAddr     string
	CoreCommandAddr      string
	EdgeSyncPeriod       uint
}

func NewYurtDeviceControllerOptions() *YurtDeviceControllerOptions {
	return &YurtDeviceControllerOptions{
		MetricsAddr:          ":8080",
		ProbeAddr:            ":8080",
		EnableLeaderElection: false,
		Nodepool:             "",
		Namespace:            "default",
		CoreDataAddr:         "edgex-core-data:48080",
		CoreMetadataAddr:     "edgex-core-metadata:48081",
		CoreCommandAddr:      "edgex-core-command:48082",
		EdgeSyncPeriod:       5,
	}
}

func ValidateOptions(options *YurtDeviceControllerOptions) error {
	if err := ValidateEdgePlatformAddress(options); err != nil {
		return err
	}
	return nil
}

func (o *YurtDeviceControllerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.MetricsAddr, "metrics-bind-address", o.MetricsAddr, "The address the metric endpoint binds to.")
	fs.StringVar(&o.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	fs.BoolVar(&o.EnableLeaderElection, "leader-elect", false, "Enable leader election for controller manager. "+"Enabling this will ensure there is only one active controller manager.")
	fs.StringVar(&o.Nodepool, "nodepool", "", "The nodePool deviceController is deployed in.(just for debugging)")
	fs.StringVar(&o.Namespace, "namespace", "default", "The cluster namespace for edge resources synchronization.")
	fs.StringVar(&o.CoreDataAddr, "core-data-address", "edgex-core-data:48080", "The address of edge core-data service.")
	fs.StringVar(&o.CoreMetadataAddr, "core-metadata-address", "edgex-core-metadata:48081", "The address of edge core-metadata service.")
	fs.StringVar(&o.CoreCommandAddr, "core-command-address", "edgex-core-command:48082", "The address of edge core-command service.")
	fs.UintVar(&o.EdgeSyncPeriod, "edge-sync-period", 5, "The period of the device management platform synchronizing the device status to the cloud.(in seconds,not less than 5 seconds)")
}

func ValidateEdgePlatformAddress(options *YurtDeviceControllerOptions) error {
	addrs := []string{options.CoreDataAddr, options.CoreMetadataAddr, options.CoreCommandAddr}
	for _, addr := range addrs {
		if addr != "" {
			if _, _, err := net.SplitHostPort(addr); err != nil {
				return fmt.Errorf("invalid address: %s", err)
			}
		}
	}
	return nil
}
