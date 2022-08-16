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
	"path/filepath"
	"testing"
	"time"

	devicev1alpha1 "github.com/openyurtio/device-controller/apis/device.openyurt.io/v1alpha1"
	"github.com/openyurtio/device-controller/cmd/yurt-device-controller/options"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

const (
	PoolName = "pool-bj"

	CommonNamespace = "default"

	ServiceName = "test-deviceservice"

	ProfileName = "test-profile"
	Model       = "OpenYurt"

	DeviceName = "test-device"

	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("../..", "config", "crd", "bases")},
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = devicev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	co := &options.YurtDeviceControllerOptions{
		MetricsAddr:          ":8080",
		ProbeAddr:            ":8080",
		EnableLeaderElection: false,
		Nodepool:             PoolName,
		Namespace:            CommonNamespace,
		CoreDataAddr:         "edgex-core-data:59880",
		CoreMetadataAddr:     "edgex-core-metadata:59881",
		CoreCommandAddr:      "edgex-core-command:59882",
		EdgeSyncPeriod:       5,
	}

	err = (&DeviceProfileReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager, co)
	Expect(err).ToNot(HaveOccurred())

	err = (&DeviceServiceReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager, co)
	Expect(err).ToNot(HaveOccurred())

	err = (&DeviceReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager, co)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
