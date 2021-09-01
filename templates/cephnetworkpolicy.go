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
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var CephNetworkPolicyTemplate = netv1.NetworkPolicy{
	Spec: netv1.NetworkPolicySpec{
		Ingress: []netv1.NetworkPolicyIngressRule{
			{
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"policy-group.network.openshift.io/host-network": "",
							},
						},
					},
				},
			},
		},
		PolicyTypes: []netv1.PolicyType{
			netv1.PolicyTypeIngress,
		},
		PodSelector: metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "app",
					Operator: metav1.LabelSelectorOpIn,
					Values: []string{
						"rook-ceph-mds",
						"rook-ceph-osd",
						"rook-ceph-mgr",
						"rook-ceph-mon",
						"rook-ceph-crashcollector",
					},
				},
			},
		},
	},
}
