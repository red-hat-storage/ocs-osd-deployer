/*


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

package templates

import (
	ocsv1 "github.com/openshift/ocs-operator/pkg/apis/ocs/v1"
	rook "github.com/rook/rook/pkg/apis/rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageClusterTemplate is the template that serves as the base for the storage clsuter deployed by the operator
var gp2 = "gp2"
var volumeModeBlock = corev1.PersistentVolumeBlock

var StorageClusterTemplate = ocsv1.StorageCluster{
	Spec: ocsv1.StorageClusterSpec{
		// The label selector is used to select only the worker nodes for
		// both labeling and scheduling.
		LabelSelector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "node-role.kubernetes.io/worker",
				Operator: metav1.LabelSelectorOpExists,
			}, {
				Key:      "node-role.kubernetes.io/infra",
				Operator: metav1.LabelSelectorOpDoesNotExist,
			}},
		},
		ManageNodes: false,
		MonPVCTemplate: &corev1.PersistentVolumeClaim{
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &gp2,
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
			},
		},
		Resources: map[string]corev1.ResourceRequirements{
			"mds": {
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("3000m"),
					"memory": resource.MustParse("8Gi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("1000m"),
					"memory": resource.MustParse("8Gi"),
				},
			},
			"mgr": {
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("1000m"),
					"memory": resource.MustParse("3Gi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("1000m"),
					"memory": resource.MustParse("3Gi"),
				},
			},
			"mon": {
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("1000m"),
					"memory": resource.MustParse("2Gi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("1000m"),
					"memory": resource.MustParse("2Gi"),
				},
			},
		},
		StorageDeviceSets: []ocsv1.StorageDeviceSet{{
			Name:  "default",
			Count: 1,
			DataPVCTemplate: corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &gp2,
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					VolumeMode: &volumeModeBlock,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": resource.MustParse("1Ti"),
						},
					},
				},
			},
			Placement: rook.Placement{},
			Portable:  true,
			Replica:   3,
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("2000m"),
					"memory": resource.MustParse("5Gi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("1000m"),
					"memory": resource.MustParse("5Gi"),
				},
			},
		}},
		MultiCloudGateway: &ocsv1.MultiCloudGatewaySpec{
			ReconcileStrategy: "ignore",
		},
	},
}
