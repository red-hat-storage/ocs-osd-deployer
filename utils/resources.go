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

package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var resourceRequirements = map[string]corev1.ResourceRequirements{
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
	"sds": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("2000m"),
			"memory": resource.MustParse("5Gi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("1000m"),
			"memory": resource.MustParse("5Gi"),
		},
	},
	"prometheus": {
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("1"),
			"memory": resource.MustParse("200Mi"),
		},
	},
	"ocs-operator": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("250m"),
			"memory": resource.MustParse("512Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("250m"),
			"memory": resource.MustParse("512Mi"),
		},
	},
	"rook-ceph-operator": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("250m"),
			"memory": resource.MustParse("512Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("250m"),
			"memory": resource.MustParse("512Mi"),
		},
	},
	"ocs-metrics-exporter": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("250m"),
			"memory": resource.MustParse("512Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("250m"),
			"memory": resource.MustParse("512Mi"),
		},
	},
	"crashcollector": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("50m"),
			"memory": resource.MustParse("20Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("50m"),
			"memory": resource.MustParse("20Mi"),
		},
	},

	// Note: These values come from the example rook-ceph-operator-config, found here:
	// https://github.com/rook/rook/blob/master/cluster/examples/kubernetes/ceph/operator.yaml#L123-L270
	"csi-provisioner": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("128Mi"),
			"cpu":    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("256Mi"),
			"cpu":    resource.MustParse("200m"),
		},
	},

	"csi-resizer": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("128Mi"),
			"cpu":    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("256Mi"),
			"cpu":    resource.MustParse("200m"),
		},
	},

	"csi-attacher": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("128Mi"),
			"cpu":    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("256Mi"),
			"cpu":    resource.MustParse("200m"),
		},
	},

	"csi-snapshotter": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("128Mi"),
			"cpu":    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("256Mi"),
			"cpu":    resource.MustParse("200m"),
		},
	},

	"csi-rbdplugin": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("512Mi"),
			"cpu":    resource.MustParse("250m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("1Gi"),
			"cpu":    resource.MustParse("500m"),
		},
	},

	"liveness-prometheus": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("128Mi"),
			"cpu":    resource.MustParse("50m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("256Mi"),
			"cpu":    resource.MustParse("100m"),
		},
	},

	"driver-registrar": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("128Mi"),
			"cpu":    resource.MustParse("50m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("256Mi"),
			"cpu":    resource.MustParse("100m"),
		},
	},

	"csi-cephfsplugin": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("512Mi"),
			"cpu":    resource.MustParse("250m"),
		},
	},
}

func GetResourceRequirements(name string) corev1.ResourceRequirements {
	if req, ok := resourceRequirements[name]; ok {
		return req
	}
	panic(fmt.Sprintf("Resource requirement not found: %v", name))
}
