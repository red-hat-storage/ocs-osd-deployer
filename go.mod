module github.com/openshift/ocs-osd-deployer

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/ocs-operator v0.0.1-alpha1.0.20201201172124-0811c33c21b2
	github.com/operator-framework/api v0.1.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v12.0.0+incompatible
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
