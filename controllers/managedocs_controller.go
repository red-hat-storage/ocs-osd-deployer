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

package controllers

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	ocsv1 "github.com/openshift/ocs-operator/pkg/apis/ocs/v1"
	v1 "github.com/openshift/ocs-osd-deployer/api/v1alpha1"
	"github.com/openshift/ocs-osd-deployer/templates"
	"github.com/openshift/ocs-osd-deployer/utils"
)

const (
	storageClusterName = "ocs-storagecluster"
)

// ManagedOCSReconciler reconciles a ManagedOCS object
type ManagedOCSReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context

	managedOCS *v1.ManagedOCS
}

// Add necessary rbac permissions for managedocs finalizer in order to set blockOwnerDeletion.
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources={managedocs,managedocs/finalizers},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=managedocs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=storageclusters,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager TODO
func (r *ManagedOCSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlOptions := controller.Options{
		MaxConcurrentReconciles: 1,
	}
	managedOCSPredicates := builder.WithPredicates(
		predicate.GenerationChangedPredicate{},
	)

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(ctrlOptions).
		For(&v1.ManagedOCS{}, managedOCSPredicates).
		Owns(&ocsv1.StorageCluster{}).
		Complete(r)
}

// Reconcile TODO
func (r *ManagedOCSReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("req.Namespace", req.Namespace, "req.Name", req.Name)
	log.Info("Reconciling ManagedOCS")

	r.ctx = context.Background()

	// Load the managed ocs resource
	r.managedOCS = &v1.ManagedOCS{}
	if err := r.Get(r.ctx, req.NamespacedName, r.managedOCS); err != nil {
		return ctrl.Result{}, err
	}

	// Run the reconcile phases
	err := r.reconcilePhases(req)

	// Ensure status is updated once even on failed reconciles
	statusErr := r.Status().Update(r.ctx, r.managedOCS)

	// Reconcile errors have priority to status update errors
	if err != nil {
		return ctrl.Result{}, err
	} else if statusErr != nil {
		return ctrl.Result{}, statusErr
	} else {
		return ctrl.Result{}, nil
	}
}

func (r *ManagedOCSReconciler) reconcilePhases(req ctrl.Request) error {
	// Set the effective reconcile strategy
	reconcileStrategy := v1.ReconcileStrategyStrict
	if strings.EqualFold(string(r.managedOCS.Spec.ReconcileStrategy), string(v1.ReconcileStrategyNone)) {
		reconcileStrategy = v1.ReconcileStrategyNone
	}
	r.managedOCS.Status.ReconcileStrategy = reconcileStrategy

	// Create or update an existing storage cluster
	storageCluster := &ocsv1.StorageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageClusterName,
			Namespace: req.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(r.ctx, r, storageCluster, func() error {
		return r.setDesiredStorageCluster(reconcileStrategy, storageCluster)
	}); err != nil {
		return err
	}

	return nil
}

// Set the desired stats for the storage cluster resource
func (r *ManagedOCSReconciler) setDesiredStorageCluster(
	reconcileStrategy v1.ReconcileStrategy,
	sc *ocsv1.StorageCluster) error {
	r.Log.Info("Reconciling storagecluster", "ReconcileStrategy", reconcileStrategy)

	// Ensure ownership on the storage cluster CR
	if err := ctrl.SetControllerReference(r.managedOCS, sc, r.Scheme); err != nil {
		return err
	}

	// Handle strict mode reconciliation
	if reconcileStrategy == v1.ReconcileStrategyStrict {
		// Get an instance of the desired state
		desired := utils.ObjectFromTemplate(templates.StorageClusterTemplate, r.Scheme).(*ocsv1.StorageCluster)

		// Override storage cluster spec with desired spec from the template.
		// We do not replace meta or status on purpose
		sc.Spec = desired.Spec
	}

	return nil
}
