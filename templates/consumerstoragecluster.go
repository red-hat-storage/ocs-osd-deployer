package templates

import (
	"k8s.io/apimachinery/pkg/api/resource"

	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v1"
)

var ConsumerStorageClusterTemplate = ocsv1.StorageCluster{
	Spec: ocsv1.StorageClusterSpec{
		ExternalStorage: ocsv1.ExternalStorageClusterSpec{
			Enable:                  true,
			StorageProviderKind:     ocsv1.KindOCS,
			StorageProviderEndpoint: "",
			OnboardingTicket:        "",
			RequestedCapacity:       &resource.Quantity{},
		},
		MultiCloudGateway: &ocsv1.MultiCloudGatewaySpec{
			ReconcileStrategy: "ignore",
		},
	},
}
