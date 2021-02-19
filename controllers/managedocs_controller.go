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
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	ocsv1 "github.com/openshift/ocs-operator/pkg/apis/ocs/v1"
	v1 "github.com/openshift/ocs-osd-deployer/api/v1alpha1"
	"github.com/openshift/ocs-osd-deployer/templates"
	"github.com/openshift/ocs-osd-deployer/utils"
	corev1 "k8s.io/api/core/v1"
)

const (
	storageClusterName  = "ocs-storagecluster"
	storageClassSizeKey = "size"
	baseOsdSize         = "1Ti"
)

// ManagedOCSReconciler reconciles a ManagedOCS object
type ManagedOCSReconciler struct {
	client.Client
	Log                  logr.Logger
	Scheme               *runtime.Scheme
	ctx                  context.Context
	managedOCS           *v1.ManagedOCS
	AddonParamSecretName string
}

// Add necessary rbac permissions for managedocs finalizer in order to set blockOwnerDeletion.
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources={managedocs,managedocs/finalizers},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=managedocs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=storageclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",namespace=system,resources=secrets,verbs=get;list;watch

// SetupWithManager TODO
func (r *ManagedOCSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mapFn := handler.ToRequestsFunc(
		func(obj handler.MapObject) []reconcile.Request {
			if obj.Meta.GetName() == r.AddonParamSecretName {
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{
						Name:      "managedocs",
						Namespace: obj.Meta.GetNamespace(),
					}},
				}
			}
			return nil
		})

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ManagedOCS{}).
		Owns(&ocsv1.StorageCluster{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn},
		).
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
	if _, err := ctrlutil.CreateOrUpdate(r.ctx, r, storageCluster, func() error {
		return r.generateStorageCluster(reconcileStrategy, storageCluster)
	}); err != nil {
		return err
	}

	return nil
}

// Set the desired stats for the storage cluster resource
func (r *ManagedOCSReconciler) generateStorageCluster(
	reconcileStrategy v1.ReconcileStrategy,
	sc *ocsv1.StorageCluster) error {
	r.Log.Info("Reconciling storagecluster", "ReconcileStrategy", reconcileStrategy)

	// Ensure ownership on the storage cluster CR
	if err := ctrlutil.SetControllerReference(r.managedOCS, sc, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference on storage cluster")
		return err
	}

	// Handle strict mode reconciliation
	if reconcileStrategy == v1.ReconcileStrategyStrict {
		// Get an instance of the desired state
		desired := utils.ObjectFromTemplate(templates.StorageClusterTemplate, r.Scheme).(*ocsv1.StorageCluster)

		if err := r.setStorageClusterParamsFromSecret(desired); err != nil {
			return err
		}
		// Override storage cluster spec with desired spec from the template.
		// We do not replace meta or status on purpose

		sc.Spec = desired.Spec
	}

	return nil
}

func (r *ManagedOCSReconciler) setStorageClusterParamsFromSecret(
	sc *ocsv1.StorageCluster) error {

	// The addon param secret will contain the capacity of the cluster in Ti
	// size = 1,  creates a cluster of 1 Ti capacity
	// size = 2,  creates a cluster of 2 Ti capacity etc

	addonParamSecret := &corev1.Secret{}
	if err := r.Get(r.ctx, types.NamespacedName{Namespace: r.managedOCS.ObjectMeta.Namespace, Name: r.AddonParamSecretName}, addonParamSecret); err != nil {
		// Do not create the StorageCluster if the we fail to get the addon param secret
		r.Log.Error(err, "Failed to get the addon param secret.", "SecretName", r.AddonParamSecretName)
		return err
	}
	data := addonParamSecret.Data

	sdsCount, err := strconv.Atoi(string(data[storageClassSizeKey]))
	if err != nil {
		r.Log.Error(err, "Invalid storage cluster size value", "Size", string(data[storageClassSizeKey]))
		return err
	}
	osdSize, _ := resource.ParseQuantity(baseOsdSize)

	r.Log.Info("Storage cluster parameters", "count", sdsCount, "osdSize", osdSize)
	sc.Spec.StorageDeviceSets[0].Count = sdsCount
	persistentVolume := &sc.Spec.StorageDeviceSets[0].DataPVCTemplate
	persistentVolume.Spec.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			"storage": osdSize,
		},
	}
	return nil
}
