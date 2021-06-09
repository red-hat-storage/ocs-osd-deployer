module github.com/openshift/ocs-osd-deployer

go 1.15

require (
	cloud.google.com/go v0.74.0 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/ocs-operator v0.0.1-alpha1.0.20201201172124-0811c33c21b2
	github.com/operator-framework/api v0.1.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.47.0
	github.com/prometheus/client_golang v1.10.0 // indirect
	github.com/prometheus/common v0.20.0 // indirect
	github.com/rook/rook v1.4.6
	github.com/stretchr/testify v1.7.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/zap v1.14.1
	golang.org/x/crypto v0.0.0-20201208171446-5f87f3452ae9 // indirect
	golang.org/x/oauth2 v0.0.0-20210210192628-66670185b0cd // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	golang.org/x/tools v0.1.0 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.21.0
	k8s.io/apiextensions-apiserver v0.21.0 // indirect
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10 // indirect
	sigs.k8s.io/controller-runtime v0.6.3
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20201203102015-275406142edb // required for Quickstart CRD
	k8s.io/api => k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.3
	k8s.io/apiserver => k8s.io/apiserver v0.19.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.3
	k8s.io/client-go => k8s.io/client-go v0.19.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.3
	k8s.io/code-generator => k8s.io/code-generator v0.19.3
	k8s.io/component-base => k8s.io/component-base v0.19.3
	k8s.io/cri-api => k8s.io/cri-api v0.19.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.3
	k8s.io/kubectl => k8s.io/kubectl v0.19.3
	k8s.io/kubelet => k8s.io/kubelet v0.19.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.3
	k8s.io/metrics => k8s.io/metrics v0.19.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.3
)
