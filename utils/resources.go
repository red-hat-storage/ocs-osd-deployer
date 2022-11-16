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
			"cpu":    resource.MustParse("1500m"),
			"memory": resource.MustParse("8Gi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("1500m"),
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
			"memory": resource.MustParse("7Gi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("1000m"),
			"memory": resource.MustParse("7Gi"),
		},
	},
	"prometheus": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("450m"),
			"memory": resource.MustParse("250Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("450m"),
			"memory": resource.MustParse("250Mi"),
		},
	},
	"alertmanager": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("100m"),
			"memory": resource.MustParse("200Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("100m"),
			"memory": resource.MustParse("200Mi"),
		},
	},
	"ocs-operator": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("200m"),
			"memory": resource.MustParse("800Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("200m"),
			"memory": resource.MustParse("800Mi"),
		},
	},
	"rook-ceph-operator": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("300m"),
			"memory": resource.MustParse("200Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("300m"),
			"memory": resource.MustParse("200Mi"),
		},
	},
	"ocs-metrics-exporter": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("60m"),
			"memory": resource.MustParse("75Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("60m"),
			"memory": resource.MustParse("75Mi"),
		},
	},
	"crashcollector": {
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("50m"),
			"memory": resource.MustParse("80Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("50m"),
			"memory": resource.MustParse("80Mi"),
		},
	},
	"csi-provisioner": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("85Mi"),
			"cpu":    resource.MustParse("15m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("85Mi"),
			"cpu":    resource.MustParse("15m"),
		},
	},

	"csi-resizer": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("55Mi"),
			"cpu":    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("55Mi"),
			"cpu":    resource.MustParse("10m"),
		},
	},

	"csi-attacher": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("45Mi"),
			"cpu":    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("45Mi"),
			"cpu":    resource.MustParse("10m"),
		},
	},

	"csi-snapshotter": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("35Mi"),
			"cpu":    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("35Mi"),
			"cpu":    resource.MustParse("10m"),
		},
	},

	"csi-rbdplugin": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("270Mi"),
			"cpu":    resource.MustParse("25m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("270Mi"),
			"cpu":    resource.MustParse("25m"),
		},
	},

	"liveness-prometheus": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("30Mi"),
			"cpu":    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("30Mi"),
			"cpu":    resource.MustParse("10m"),
		},
	},

	"driver-registrar": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("25Mi"),
			"cpu":    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("25Mi"),
			"cpu":    resource.MustParse("10m"),
		},
	},

	"csi-cephfsplugin": {
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("160Mi"),
			"cpu":    resource.MustParse("20m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("160Mi"),
			"cpu":    resource.MustParse("20m"),
		},
	},

	"kube-rbac-proxy": {
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("30Mi"),
			"cpu":    resource.MustParse("50m"),
		},
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("30Mi"),
			"cpu":    resource.MustParse("50m"),
		},
	},
}

func GetResourceRequirements(name string) corev1.ResourceRequirements {
	if req, ok := resourceRequirements[name]; ok {
		return req
	}
	panic(fmt.Sprintf("Resource requirement not found: %v", name))
}
