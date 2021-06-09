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

// This prometheus rule ensures that a DMS alert occurs during every prometheus scrape.
var DMSPrometheusRuleTemplate = promv1.PrometheusRule{
	Spec: promv1.PrometheusRuleSpec{
		Groups: []promv1.RuleGroup{
			{
				Name: "snitch-alert",
				Rules: []promv1.Rule{
					{
						Alert: "DeadMansSnitch",
						Expr: intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "vector(1)",
						},
						Labels: map[string]string{
							"alertname": "DeadMansSnitch",
						},
					},
				},
			},
		},
	},
}
