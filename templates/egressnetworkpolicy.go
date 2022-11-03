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
	openshiftv1 "github.com/openshift/api/network/v1"
)

var EgressNetworkPolicyTemplate = openshiftv1.EgressNetworkPolicy{
	Spec: openshiftv1.EgressNetworkPolicySpec{
		Egress: []openshiftv1.EgressNetworkPolicyRule{
			{
				To: openshiftv1.EgressNetworkPolicyPeer{
					DNSName: "events.pagerduty.com",
				},
				Type: openshiftv1.EgressNetworkPolicyRuleAllow,
			},
			{
				To: openshiftv1.EgressNetworkPolicyPeer{
					CIDRSelector: "100.64.0.0/16",
				},
				Type: openshiftv1.EgressNetworkPolicyRuleAllow,
			},
			{
				To: openshiftv1.EgressNetworkPolicyPeer{
					CIDRSelector: "0.0.0.0/0",
				},
				Type: openshiftv1.EgressNetworkPolicyRuleDeny,
			},
		},
	},
}
