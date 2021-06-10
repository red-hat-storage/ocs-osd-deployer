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
	"encoding/json"

	promv1a1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func convertToApiExtV1JSON(val interface{}) apiextensionsv1.JSON {
	raw, err := json.Marshal(val)
	if err != nil {
		panic(err)
	}

	out := apiextensionsv1.JSON{}
	out.Raw = raw
	return out
}

var match1 = []promv1a1.Matcher{{Name: "severity"}, {Value: "critical"}}

var AlertmanagerConfigTemplate = promv1a1.AlertmanagerConfig{
	Spec: promv1a1.AlertmanagerConfigSpec{
		Route: &promv1a1.Route{
			Routes: []apiextensionsv1.JSON{
				convertToApiExtV1JSON(promv1a1.Route{
					GroupBy:        []string{"alertname"},
					GroupWait:      "30s",
					GroupInterval:  "5m",
					RepeatInterval: "12h",
					Matchers:       []promv1a1.Matcher{{Name: "severity", Value: "critical"}},
					Receiver:       "pagerduty",
				},
				),
				convertToApiExtV1JSON(promv1a1.Route{
					GroupBy:        []string{"alertname"},
					GroupWait:      "30s",
					GroupInterval:  "5m",
					RepeatInterval: "5m",
					Matchers:       []promv1a1.Matcher{{Name: "alertname", Value: "DeadMansSnitch"}},
					Receiver:       "DeadMansSnitch",
				},
				),
			},
		},
		Receivers: []promv1a1.Receiver{{
			Name: "pagerduty",
			PagerDutyConfigs: []promv1a1.PagerDutyConfig{{
				ServiceKey: &corev1.SecretKeySelector{Key: ""},
				Details:    []promv1a1.KeyValue{{Key: "", Value: ""}},
			}},
		}, {
			Name:           "DeadMansSnitch",
			WebhookConfigs: []promv1a1.WebhookConfig{{}}}},
	},
}
