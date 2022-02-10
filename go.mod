module github.com/openyurtio/device-controller

go 1.15

require (
	github.com/edgexfoundry/go-mod-core-contracts v0.1.111
	github.com/go-resty/resty/v2 v2.4.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20210428140749-89ef3d95e781
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/cluster-api v0.4.2
	sigs.k8s.io/controller-runtime v0.9.6
)
