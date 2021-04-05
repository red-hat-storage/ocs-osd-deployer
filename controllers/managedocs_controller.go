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
	"fmt"
	"strconv"
	"strings"
	"time"

	operators "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	ocsv1 "github.com/openshift/ocs-operator/pkg/apis/ocs/v1"
	v1 "github.com/openshift/ocs-osd-deployer/api/v1alpha1"
	"github.com/openshift/ocs-osd-deployer/templates"
	"github.com/openshift/ocs-osd-deployer/utils"
)

const (
	storageClusterName     = "ocs-storagecluster"
	storageClassSizeKey    = "size"
	deviceSetName          = "default"
	storageClassRbdName    = "ocs-storagecluster-ceph-rbd"
	storageClassCephFsName = "ocs-storagecluster-cephfs"
	deployerCsvPrefix      = "ocs-osd-deployer"
	ManagedOcsFinalizer    = "managedocs.ocs.openshift.io"
)

// ManagedOCSReconciler reconciles a ManagedOCS object
type ManagedOCSReconciler struct {
	client.Client
	UnrestrictedClient      client.Client
	Log                     logr.Logger
	Scheme                  *runtime.Scheme
	AddonParamSecretName    string
	DeleteConfigMapName     string
	DeleteConfigMapLabelKey string
	AddonSubscriptionName   string

	ctx        context.Context
	managedOCS *v1.ManagedOCS
	namespace  string
}

// Add necessary rbac permissions for managedocs finalizer in order to set blockOwnerDeletion.
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources={managedocs,managedocs/finalizers},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=managedocs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=storageclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",namespace=system,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=operators.coreos.com,namespace=system,resources={subscriptions,clusterserviceversions},verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch

// SetupWithManager TODO
func (r *ManagedOCSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlOptions := controller.Options{
		MaxConcurrentReconciles: 1,
	}
	managedOCSPredicates := builder.WithPredicates(
		predicate.GenerationChangedPredicate{},
	)
	addonParamsSecretWatchHandler := handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(
			func(obj handler.MapObject) []reconcile.Request {
				if obj.Meta.GetName() == r.AddonParamSecretName {
					return []reconcile.Request{{
						NamespacedName: types.NamespacedName{
							Name:      "managedocs",
							Namespace: obj.Meta.GetNamespace(),
						},
					}}
				}
				return nil
			},
		),
	}
	deleteLabelWatchHandler := handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(
			func(obj handler.MapObject) []reconcile.Request {
				if obj.Meta.GetName() == r.DeleteConfigMapName {
					if _, ok := obj.Meta.GetLabels()[r.DeleteConfigMapLabelKey]; ok {
						return []reconcile.Request{{
							NamespacedName: types.NamespacedName{
								Name:      "managedocs",
								Namespace: obj.Meta.GetNamespace(),
							},
						}}
					}
				}
				return nil
			},
		),
	}

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(ctrlOptions).
		For(&v1.ManagedOCS{}, managedOCSPredicates).
		Owns(&ocsv1.StorageCluster{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &addonParamsSecretWatchHandler).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, &deleteLabelWatchHandler).
		Complete(r)
}

// Reconcile TODO
func (r *ManagedOCSReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("req.Namespace", req.Namespace, "req.Name", req.Name)
	log.Info("Reconciling ManagedOCS")

	r.ctx = context.Background()
	r.namespace = req.Namespace

	// Load the managed ocs resource
	r.managedOCS = &v1.ManagedOCS{}
	if err := r.Get(r.ctx, req.NamespacedName, r.managedOCS); err != nil {
		if errors.IsNotFound(err) {
			r.Log.V(-1).Info("ManagedOCS resource not found")
		} else {
			return ctrl.Result{}, err
		}
	}

	// Run the reconcile phases
	result, err := r.reconcilePhases()
	if err != nil {
		r.Log.Error(err, "An error was encountered during reconcilePhases")
	}

	// Ensure status is updated once even on failed reconciles
	var statusErr error
	if r.managedOCS.UID != "" {
		statusErr = r.Status().Update(r.ctx, r.managedOCS)
	}

	// Reconcile errors have priority to status update errors
	if err != nil {
		return ctrl.Result{}, err
	} else if statusErr != nil {
		return ctrl.Result{}, statusErr
	} else {
		return result, nil
	}
}

func (r *ManagedOCSReconciler) reconcilePhases() (reconcile.Result, error) {
	// Update the status of the components
	r.managedOCS.Status.Components = r.updateComponents()

	if !r.managedOCS.DeletionTimestamp.IsZero() {

		err := r.deleteComponents()
		if err != nil {
			return ctrl.Result{}, err
		}

		if r.verifyComponentsDoNotExists() {
			r.Log.Info("Removing finalizer from managedOCS")
			r.managedOCS.ObjectMeta.Finalizers = utils.Remove(r.managedOCS.ObjectMeta.Finalizers, ManagedOcsFinalizer)
			if err := r.Client.Update(r.ctx, r.managedOCS); err != nil {
				return ctrl.Result{}, fmt.Errorf("Failed to remove finalizer from managedOCS: %v", err)
			}
			r.Log.Info("managedOCS removed successfully")
		}
	} else if r.managedOCS.UID != "" {

		if !utils.Contains(r.managedOCS.GetFinalizers(), ManagedOcsFinalizer) {
			r.Log.V(-1).Info("Finalizer not found for managedOCS. Adding finalizer")
			r.managedOCS.ObjectMeta.Finalizers = append(r.managedOCS.ObjectMeta.Finalizers, ManagedOcsFinalizer)
			if err := r.Client.Update(r.ctx, r.managedOCS); err != nil {
				return ctrl.Result{}, fmt.Errorf("Failed to update managedOCS with finalizer: %v", err)
			}
		}
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
				Namespace: r.namespace,
			},
		}
		if _, err := ctrl.CreateOrUpdate(r.ctx, r, storageCluster, func() error {
			return r.generateStorageCluster(reconcileStrategy, storageCluster)
		}); err != nil {
			return ctrl.Result{}, err
		}

		if r.checkUninstallCondition() && r.areComponentsReadyForUninstall() {
			found, err := r.findOcsPvcs()
			if err != nil {
				return ctrl.Result{}, err
			}
			if found {
				r.Log.Info("Found consumer PVCs using OCS storageclasses, cannot proceed on uninstallation")
				return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
			}
			r.Log.Info("Starting OCS uninstallation")

			r.Log.Info("Deleting managedOCS")
			if err := r.Client.Delete(r.ctx, r.managedOCS); err != nil && !errors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf("Unable to delete managedocs: %v", err)
			}
		}
	} else if r.checkUninstallCondition() {

		if err := r.removeOlmComponents(); err != nil {
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

// Set the desired stats for the storage cluster resource
func (r *ManagedOCSReconciler) generateStorageCluster(
	reconcileStrategy v1.ReconcileStrategy,
	sc *ocsv1.StorageCluster,
) error {
	r.Log.Info("Reconciling storagecluster", "ReconcileStrategy", reconcileStrategy)

	// Ensure ownership on the storage cluster CR
	if err := ctrl.SetControllerReference(r.managedOCS, sc, r.Scheme); err != nil {
		return err
	}

	// Get sds count of storage cluster
	currSdsCount := 0
	for _, item := range sc.Spec.StorageDeviceSets {
		if item.Name == deviceSetName {
			currSdsCount = item.Count
			break
		}
	}

	// Handle strict mode reconciliation
	if reconcileStrategy == v1.ReconcileStrategyStrict {
		// Get an instance of the desired state
		desired := utils.ObjectFromTemplate(templates.StorageClusterTemplate, r.Scheme).(*ocsv1.StorageCluster)
		if err := r.setStorageClusterParamsFromSecret(desired, currSdsCount); err != nil {
			return err
		}

		// Override storage cluster spec with desired spec from the template.
		// We do not replace meta or status on purpose
		sc.Spec = desired.Spec
	}
	return nil
}

func (r *ManagedOCSReconciler) setStorageClusterParamsFromSecret(sc *ocsv1.StorageCluster, currSdsCount int) error {

	// The addon param secret will contain the capacity of the cluster in Ti
	// size = 1,  creates a cluster of 1 Ti capacity
	// size = 2,  creates a cluster of 2 Ti capacity etc

	addonParamSecret := corev1.Secret{}
	if err := r.Get(
		r.ctx,
		types.NamespacedName{Namespace: r.namespace, Name: r.AddonParamSecretName},
		&addonParamSecret,
	); err != nil {
		// Do not create the StorageCluster if the we fail to get the addon param secret
		return fmt.Errorf("Failed to get the addon param secret, Secret Name: %v", r.AddonParamSecretName)
	}
	addonParams := addonParamSecret.Data

	sizeAsString := string(addonParams[storageClassSizeKey])
	r.Log.Info("Requested add-on settings", storageClassSizeKey, sizeAsString)
	sdsCount, err := strconv.Atoi(sizeAsString)
	if err != nil {
		return fmt.Errorf("Invalid storage cluster size value: %v", sizeAsString)
	}

	var ds *ocsv1.StorageDeviceSet = nil
	for index := range sc.Spec.StorageDeviceSets {
		item := &sc.Spec.StorageDeviceSets[index]
		if item.Name == deviceSetName {
			ds = item
			break
		}
	}
	if ds == nil {
		return fmt.Errorf("cloud not find default device set on stroage cluster")
	}

	// Prevent downscaling by comparing count from secret and count from storage cluster
	r.Log.Info("Setting storage device set count", "Current", currSdsCount, "New", sdsCount)
	if currSdsCount <= sdsCount {
		ds.Count = sdsCount
	} else {
		r.Log.V(-1).Info("Requested storage device set count will result in downscaling, which is not supported. Skipping")
		ds.Count = currSdsCount
	}

	return nil
}

func (r *ManagedOCSReconciler) updateComponents() v1.ComponentStatusMap {
	var components v1.ComponentStatusMap

	// Checking the StorageCluster component.
	var sc ocsv1.StorageCluster

	scNamespacedName := types.NamespacedName{
		Name:      storageClusterName,
		Namespace: r.namespace,
	}

	err := r.Get(r.ctx, scNamespacedName, &sc)
	if err == nil {
		if sc.Status.Phase == "Ready" {
			components.StorageCluster.State = v1.ComponentReady
		} else {
			components.StorageCluster.State = v1.ComponentPending
		}
	} else if errors.IsNotFound(err) {
		components.StorageCluster.State = v1.ComponentNotFound
	} else {
		r.Log.Error(err, "error getting StorageCluster, setting compoment status to Unknown")
		components.StorageCluster.State = v1.ComponentUnknown
	}

	return components
}

func (r *ManagedOCSReconciler) checkUninstallCondition() bool {
	cmNamespacedName := types.NamespacedName{
		Name:      r.DeleteConfigMapName,
		Namespace: r.namespace,
	}

	configmap := &corev1.ConfigMap{}
	err := r.Get(r.ctx, cmNamespacedName, configmap)
	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		r.Log.Error(err, "Unable to get configMap")
		return false
	}
	_, ok := configmap.Labels[r.DeleteConfigMapLabelKey]
	return ok
}

func (r *ManagedOCSReconciler) removeOlmComponents() error {

	r.Log.Info("Deleting subscription")
	subscription := &operators.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.AddonSubscriptionName,
			Namespace: r.namespace,
		},
	}
	if err := r.Client.Delete(r.ctx, subscription); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Unable to delete subscription: %v", err)
	}
	r.Log.Info("subscription removed successfully")

	r.Log.Info("Deleting CSV")
	csvList := &operators.ClusterServiceVersionList{}
	err := r.Client.List(r.ctx, csvList)
	if err != nil {
		r.Log.Error(err, "Unable to list CSVs")
		return err
	}
	var csv *operators.ClusterServiceVersion = nil
	for index := range csvList.Items {
		candidate := &csvList.Items[index]
		if strings.HasPrefix(candidate.Name, deployerCsvPrefix) {
			csv = candidate
			break
		}
	}
	if csv != nil {
		if err := r.Client.Delete(r.ctx, csv); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("Unable to delete csv: %v", err)
		}
	}
	r.Log.Info("CSV removed successfully")

	return nil
}

func (r *ManagedOCSReconciler) areComponentsReadyForUninstall() bool {
	subComponent := r.managedOCS.Status.Components

	if subComponent.StorageCluster.State != v1.ComponentReady {
		return false
	}
	return true
}

func (r *ManagedOCSReconciler) deleteComponents() error {
	r.Log.Info("Deleting storageCluster")
	storageCluster := &ocsv1.StorageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageClusterName,
			Namespace: r.namespace,
		},
	}
	if err := r.Client.Delete(r.ctx, storageCluster); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Unable to delete storagecluster: %v", err)
	}
	return nil
}

func (r *ManagedOCSReconciler) verifyComponentsDoNotExists() bool {
	subComponent := r.managedOCS.Status.Components

	if subComponent.StorageCluster.State == v1.ComponentNotFound {
		return true
	}
	return false
}

func (r *ManagedOCSReconciler) findOcsPvcs() (bool, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	err := r.UnrestrictedClient.List(r.ctx, pvcList)
	if err != nil {
		return false, fmt.Errorf("Unable to list PVCs: %v", err)
	}
	for _, pvc := range pvcList.Items {
		scName := *pvc.Spec.StorageClassName
		if scName == storageClassCephFsName || scName == storageClassRbdName {
			return true, nil
		}
	}
	return false, nil
}
