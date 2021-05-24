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
	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/openshift/ocs-osd-deployer/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// PrometheusTemplate is the template that serves as the base for the prometheus deployed by the operator
var resourceSelector = metav1.LabelSelector{
	MatchLabels: map[string]string{
		"app": "managed-ocs",
	},
}

var PrometheusTemplate = promv1.Prometheus{
	Spec: promv1.PrometheusSpec{
		ServiceAccountName:     "prometheus-k8s",
		ServiceMonitorSelector: &resourceSelector,
		PodMonitorSelector:     &resourceSelector,
		RuleSelector:           &resourceSelector,
		EnableAdminAPI:         false,
		Alerting: &promv1.AlertingSpec{
			Alertmanagers: []promv1.AlertmanagerEndpoints{{
				Namespace: "",
				Name:      "alertmanager-operated",
				Port:      intstr.FromString("web"),
			}},
		},
		Resources: utils.GetResourceRequirements("prometheus"),
	},
}
