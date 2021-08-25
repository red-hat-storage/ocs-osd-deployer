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
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Prom rule to raise an alert if PVCs are using OCS storage classes and uninstall has been initiated
var AdditionalPrometheusRule = promv1.PrometheusRule{
	Spec: promv1.PrometheusRuleSpec{
		Groups: []promv1.RuleGroup{
			{
				Name: "uninstall-stuck-pvc-present",
				Rules: []promv1.Rule{
					{
						Alert: "UninstallStuckDueToPVC",
						Expr: intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "count(kube_persistentvolumeclaim_info * on (storageclass)  group_left(provisioner) kube_storageclass_info {provisioner=~'(.*rbd.csi.ceph.com)|(.*cephfs.csi.ceph.com)'}) * max(odfms_phase{phase='Uninstalling'}) > 0",
						},
						Labels: map[string]string{
							"alertname": "UninstallStuckDueToPVC",
							"severity":  "warning",
						},
						Annotations: map[string]string{
							"description": "Found consumer PVCs using OCS storageclasses, cannot proceed on uninstallation",
						},
					},
				},
			},
		},
	},
}
