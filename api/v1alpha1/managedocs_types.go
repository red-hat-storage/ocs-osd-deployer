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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReconcileStrategy represent the action the deployer should take whenever a recncile event occures
type ReconcileStrategy string

const (
	// ReconcileStrategyNone is used to indicate that the deployer should not
	// touch the storage cluster spec
	ReconcileStrategyNone ReconcileStrategy = "none"

	// ReconcileStrategyStrict is used to indicate that the deployer should enforce
	// storage clsuter based on a predefined spec
	ReconcileStrategyStrict ReconcileStrategy = "strict"
)

// ManagedOCSSpec defines the desired state of ManagedOCS
type ManagedOCSSpec struct {
	ReconcileStrategy              ReconcileStrategy `json:"reconcileStrategy,omitempty"`
	NetworkPolicyReconcileStrategy ReconcileStrategy `json:"networkPolicyReconcileStrategy,omitempty"`
}

type ComponentState string

const (
	ComponentReady    ComponentState = "Ready"
	ComponentPending  ComponentState = "Pending"
	ComponentNotFound ComponentState = "NotFound"
	ComponentUnknown  ComponentState = "Unknown"
)

type ComponentStatus struct {
	State ComponentState `json:"state"`
}

type ComponentStatusMap struct {
	StorageCluster ComponentStatus `json:"storageCluster"`
	Prometheus     ComponentStatus `json:"prometheus"`
	Alertmanager   ComponentStatus `json:"alertmanager"`
}

// ManagedOCSStatus defines the observed state of ManagedOCS
type ManagedOCSStatus struct {
	ReconcileStrategy              ReconcileStrategy  `json:"reconcileStrategy,omitempty"`
	Components                     ComponentStatusMap `json:"components"`
	NetworkPolicyReconcileStrategy ReconcileStrategy  `json:"networkPolicyReconcileStrategy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mocs

// ManagedOCS is the Schema for the managedocs API
type ManagedOCS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedOCSSpec   `json:"spec,omitempty"`
	Status ManagedOCSStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedOCSList contains a list of ManagedOCS
type ManagedOCSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedOCS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedOCS{}, &ManagedOCSList{})
}
