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

// AlertmanagerConfigTemplate is the template that serves as the base for the Alert Manager Configuration
// deployed by the operator
const AlertmanagerConfigTemplate = `
apiVersion: monitoring.coreos.com/v1alpha1
kind: AlertmanagerConfig
spec:
  route:
    group_wait: 30s
    group_interval: 5m
    repeat_interval: 4h
    routes:
      - match:
          severity: critical
        receiver: pagerduty
  receivers:
    - name: pagerduty
      pagerduty_configs:
        - service_key: ""
`
