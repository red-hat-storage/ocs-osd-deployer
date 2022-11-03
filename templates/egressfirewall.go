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
	ovnv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressfirewall/v1"
)

var EgressFirewallTemplate = ovnv1.EgressFirewall{
	Spec: ovnv1.EgressFirewallSpec{
		Egress: []ovnv1.EgressFirewallRule{
			{
				To: ovnv1.EgressFirewallDestination{
					DNSName: "events.pagerduty.com",
				},
				Type: ovnv1.EgressFirewallRuleAllow,
			},
			{
				To: ovnv1.EgressFirewallDestination{
					CIDRSelector: "100.64.0.0/16",
				},
				Type: ovnv1.EgressFirewallRuleAllow,
			},
			{
				To: ovnv1.EgressFirewallDestination{
					CIDRSelector: "0.0.0.0/0",
				},
				Type: ovnv1.EgressFirewallRuleDeny,
			},
		},
	},
}
