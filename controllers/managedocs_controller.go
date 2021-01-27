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

	"github.com/go-logr/logr"
	ocsv1 "github.com/openshift/ocs-operator/pkg/apis/ocs/v1"
	v1 "github.com/openshift/ocs-osd-deployer/api/v1alpha1"
	"github.com/openshift/ocs-osd-deployer/templates"
	"github.com/openshift/ocs-osd-deployer/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
)

const (
	storageClusterName = "ocs-storagecluster"
)

// AddonSecretName is the secret name
const (
	AddonSecretName string = "addon-ocs-converged-parameters"
)

const (
	storageClassSizeKey string = "size"
)

// ManagedOCSReconciler reconciles a ManagedOCS object
type ManagedOCSReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context

	managedOCS       *v1.ManagedOCS
	ConfigSecretName string
}

// +kubebuilder:rbac:groups=ocs.openshift.io,resources=managedocs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocs.openshift.io,resources=managedocs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager TODO
func (r *ManagedOCSReconciler) SetupWithManager(mgr ctrl.Manager) error {

	mapFn := handler.ToRequestsFunc(
		func(obj handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      "managedocs",
					Namespace: obj.Meta.GetNamespace(),
				}},
			}
		})

	/* 	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// The object name is not the secret name, so the event will be
			// ignored.
			name := e.MetaOld.GetName()
			if name != AddonSecretName {
				return false
			}
			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			name := e.Meta.GetName()
			if name != AddonSecretName {
				return false
			}
			return true
		},
	} */

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ManagedOCS{}).
		Owns(&ocsv1.StorageCluster{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn},
		//p,
		).
		Complete(r)
}

// Reconcile TODO
func (r *ManagedOCSReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("req.Namespace", req.Namespace, "req.Name", req.Name)
	log.Info("Reconciling ManagedOCS")

	var err error
	r.ctx = context.Background()

	// Load the managed ocs resource
	r.managedOCS = &v1.ManagedOCS{}
	if err := r.Get(r.ctx, req.NamespacedName, r.managedOCS); err != nil {
		return ctrl.Result{}, err
	}

	// Run the reconcile phases
	err = r.reconcilePhases(req)

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

	configSecret := &corev1.Secret{}
	err := r.Get(r.ctx, types.NamespacedName{Namespace: req.Namespace, Name: r.ConfigSecretName}, configSecret)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		} else {
			configSecret = nil
		}
	}
	// Create or update an existing storage cluster
	storageCluster := &ocsv1.StorageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageClusterName,
			Namespace: req.Namespace,
		},
	}
	if _, err := ctrlutil.CreateOrUpdate(r.ctx, r, storageCluster, func() error {
		return r.setDesiredStorageCluster(reconcileStrategy, storageCluster, configSecret)
	}); err != nil {
		return err
	}

	return nil
}

// Set the desired stats for the storage cluster resource
func (r *ManagedOCSReconciler) setDesiredStorageCluster(
	reconcileStrategy v1.ReconcileStrategy,
	sc *ocsv1.StorageCluster,
	cs *corev1.Secret) error {
	r.Log.Info("Reconciling storagecluster", "ReconcileStrategy", reconcileStrategy)

	// Ensure ownership on the storage cluster CR
	if err := ctrlutil.SetControllerReference(r.managedOCS, sc, r.Scheme); err != nil {
		return err
	}

	// Handle strict mode reconciliation
	if reconcileStrategy == v1.ReconcileStrategyStrict {
		// Get an instance of the desired state
		desired := utils.ObjectFromTemplate(templates.StorageClusterTemplate, r.Scheme).(*ocsv1.StorageCluster)
		r.updateStorageClusterWithParams(desired, cs)
		// Override storage cluster spec with desired spec from the template.
		// We do not replace meta or status on purpose
		sc.Spec = desired.Spec
		r.Log.Info("StorageCluster", "desiredSpec", desired.Spec)
		r.Log.Info("StorageCluster", "Current spec", sc.Spec)
	}

	return nil
}

func (r *ManagedOCSReconciler) updateStorageClusterWithParams(
	sc *ocsv1.StorageCluster,
	cs *corev1.Secret) error {

	if cs == nil {
		return nil
	}
	r.Log.Info("Updating storagecluster")

	sdata := cs.Data

	for key, element := range sdata {
		r.Log.Info("Secret keys:", "key", key, "value", string(element))
		PersistentVolume := &sc.Spec.StorageDeviceSets[0].DataPVCTemplate
		if isValidStorageClusterParam(key, string(sdata[key])) {
			PersistentVolume.Spec.Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": resource.MustParse(string(sdata[key])),
				},
			}
		}
		r.Log.Info("Post updation Persistent Volume ", "PV", PersistentVolume)
	}

	return nil
}

func isValidStorageClusterParam(key string, val string) bool {

	if key == storageClassSizeKey {
		if val != "1Ti" && val != "4Ti" {
			return false
		}
		return true
	}
	return false
}
