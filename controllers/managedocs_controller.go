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
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	opv1a1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	openshiftv1 "github.com/openshift/api/network/v1"
	ovnv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressfirewall/v1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1a1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	odfv1a1 "github.com/red-hat-data-services/odf-operator/api/v1alpha1"
	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v1"
	ocsv1alpha1 "github.com/red-hat-storage/ocs-operator/api/v1alpha1"
	v1 "github.com/red-hat-storage/ocs-osd-deployer/api/v1alpha1"
	"github.com/red-hat-storage/ocs-osd-deployer/templates"
	"github.com/red-hat-storage/ocs-osd-deployer/utils"
	netv1 "k8s.io/api/networking/v1"
)

const (
	ManagedOCSFinalizer    = "managedocs.ocs.openshift.io"
	EgressFirewallCRD      = "egressfirewalls.k8s.ovn.org"
	EgressNetworkPolicyCRD = "egressnetworkpolicies.network.openshift.io"

	managedOCSName                       = "managedocs"
	storageClusterName                   = "ocs-storagecluster"
	prometheusName                       = "managed-ocs-prometheus"
	prometheusServiceName                = "prometheus"
	alertmanagerName                     = "managed-ocs-alertmanager"
	alertmanagerConfigName               = "managed-ocs-alertmanager-config"
	dmsRuleName                          = "dms-monitor-rule"
	storageSizeKey                       = "size"
	storageUnitKey                       = "unit"
	onboardingTicketKey                  = "onboarding-ticket"
	storageProviderEndpointKey           = "storage-provider-endpoint"
	enableMCGKey                         = "enable-mcg"
	notificationEmailKeyPrefix           = "notification-email"
	onboardingValidationKey              = "onboarding-validation-key"
	deviceSetName                        = "default"
	storageClassRbdName                  = "ocs-storagecluster-ceph-rbd"
	storageClassCephFSName               = "ocs-storagecluster-cephfs"
	deployerCSVPrefix                    = "ocs-osd-deployer"
	ocsOperatorName                      = "ocs-operator"
	mcgOperatorName                      = "mcg-operator"
	egressNetworkPolicyName              = "egress-rule"
	egressFirewallName                   = "default"
	ingressNetworkPolicyName             = "ingress-rule"
	cephIngressNetworkPolicyName         = "ceph-ingress-rule"
	providerApiServerNetworkPolicyName   = "provider-api-server-rule"
	prometheusProxyNetworkPolicyName     = "prometheus-proxy-rule"
	monLabelKey                          = "app"
	monLabelValue                        = "managed-ocs"
	rookConfigMapName                    = "rook-ceph-operator-config"
	k8sMetricsServiceMonitorName         = "k8s-metrics-service-monitor"
	alertRelabelConfigSecretName         = "managed-ocs-alert-relabel-config-secret"
	alertRelabelConfigSecretKey          = "alertrelabelconfig.yaml"
	onboardingValidationKeySecretName    = "onboarding-ticket-key"
	convergedDeploymentType              = "converged"
	consumerDeploymentType               = "consumer"
	providerDeploymentType               = "provider"
	rhobsRemoteWriteConfigIdSecretKey    = "prom-remote-write-config-id"
	rhobsRemoteWriteConfigSecretName     = "prom-remote-write-config-secret"
	csiKMSConnectionDetailsConfigMapName = "csi-kms-connection-details"
	desiredODFSubscriptionChannel        = "stable-4.11"
	odfOLMPackageName                    = "odf-operator"
)

// ManagedOCSReconciler reconciles a ManagedOCS object
type ManagedOCSReconciler struct {
	Client             client.Client
	UnrestrictedClient client.Client
	Log                logr.Logger
	Scheme             *runtime.Scheme

	AddonParamSecretName         string
	AddonConfigMapName           string
	AddonConfigMapDeleteLabelKey string
	PagerdutySecretName          string
	DeadMansSnitchSecretName     string
	SMTPSecretName               string
	SOPEndpoint                  string
	AlertSMTPFrom                string
	CustomerNotificationHTMLPath string
	DeploymentType               string
	RHOBSSecretName              string
	RHOBSEndpoint                string
	RHSSOTokenEndpoint           string
	AvailableCRDs                map[string]bool

	ctx                              context.Context
	managedOCS                       *v1.ManagedOCS
	storageCluster                   *ocsv1.StorageCluster
	egressNetworkPolicy              *openshiftv1.EgressNetworkPolicy
	egressFirewall                   *ovnv1.EgressFirewall
	ingressNetworkPolicy             *netv1.NetworkPolicy
	cephIngressNetworkPolicy         *netv1.NetworkPolicy
	providerAPIServerNetworkPolicy   *netv1.NetworkPolicy
	prometheusProxyNetworkPolicy     *netv1.NetworkPolicy
	prometheus                       *promv1.Prometheus
	prometheusService                *corev1.Service
	dmsRule                          *promv1.PrometheusRule
	alertmanager                     *promv1.Alertmanager
	pagerdutySecret                  *corev1.Secret
	deadMansSnitchSecret             *corev1.Secret
	smtpSecret                       *corev1.Secret
	alertmanagerConfig               *promv1a1.AlertmanagerConfig
	alertRelabelConfigSecret         *corev1.Secret
	k8sMetricsServiceMonitor         *promv1.ServiceMonitor
	namespace                        string
	reconcileStrategy                v1.ReconcileStrategy
	addonParams                      map[string]string
	onboardingValidationKeySecret    *corev1.Secret
	prometheusKubeRBACConfigMap      *corev1.ConfigMap
	rhobsRemoteWriteConfigSecret     *corev1.Secret
	csiKMSConnectionDetailsConfigMap *corev1.ConfigMap
}

// Add necessary rbac permissions for managedocs finalizer in order to set blockOwnerDeletion.
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources={managedocs,managedocs/finalizers},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=managedocs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=storageclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=ocsinitializations,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=ocs.openshift.io,namespace=system,resources=storageconsumers,verbs=get;list;watch
// +kubebuilder:rbac:groups="monitoring.coreos.com",namespace=system,resources={alertmanagers,prometheuses,alertmanagerconfigs},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="monitoring.coreos.com",namespace=system,resources=prometheusrules,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="monitoring.coreos.com",namespace=system,resources=podmonitors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="monitoring.coreos.com",namespace=system,resources=servicemonitors,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups="",namespace=system,resources=secrets,verbs=create;get;list;watch;update
// +kubebuilder:rbac:groups=operators.coreos.com,namespace=system,resources=clusterserviceversions,verbs=get;list;watch;delete;update;patch
// +kubebuilder:rbac:groups="operators.coreos.com",resources=subscriptions,verbs=get;watch;list;update
// +kubebuilder:rbac:groups="apps",namespace=system,resources=statefulsets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources={persistentvolumes,secrets},verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources={services},verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="storage.k8s.io",resources=storageclass,verbs=get;list;watch
// +kubebuilder:rbac:groups="networking.k8s.io",namespace=system,resources=networkpolicies,verbs=create;get;list;watch;update
// +kubebuilder:rbac:groups="network.openshift.io",namespace=system,resources=egressnetworkpolicies,verbs=create;get;list;watch;update
// +kubebuilder:rbac:groups="k8s.ovn.org",namespace=system,resources=egressfirewalls,verbs=create;get;list;watch;update
// +kubebuilder:rbac:groups="coordination.k8s.io",namespace=system,resources=leases,verbs=create;get;list;watch;update
// +kubebuilder:rbac:groups="odf.openshift.io",namespace=system,resources=storagesystems,verbs=list;watch;delete
// +kubebuilder:rbac:groups="config.openshift.io",resources=clusterversions,verbs=get;watch;list
// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources=customresourcedefinitions,verbs=get;watch;list

// SetupWithManager creates an setup a ManagedOCSReconciler to work with the provided manager
func (r *ManagedOCSReconciler) SetupWithManager(mgr ctrl.Manager, ctrlOptions *controller.Options) error {
	if ctrlOptions == nil {
		ctrlOptions = &controller.Options{
			MaxConcurrentReconciles: 1,
		}
	}
	managedOCSPredicates := builder.WithPredicates(
		predicate.GenerationChangedPredicate{},
	)
	secretPredicates := builder.WithPredicates(
		predicate.NewPredicateFuncs(
			func(client client.Object) bool {
				name := client.GetName()
				return name == r.AddonParamSecretName ||
					name == r.PagerdutySecretName ||
					name == r.DeadMansSnitchSecretName ||
					name == r.SMTPSecretName ||
					name == r.RHOBSSecretName
			},
		),
	)
	configMapPredicates := builder.WithPredicates(
		predicate.NewPredicateFuncs(
			func(client client.Object) bool {
				name := client.GetName()
				if name == r.AddonConfigMapName {
					if _, ok := client.GetLabels()[r.AddonConfigMapDeleteLabelKey]; ok {
						return true
					}
				} else if name == rookConfigMapName {
					return true
				} else if name == utils.IMDSConfigMapName {
					return true
				} else if name == csiKMSConnectionDetailsConfigMapName {
					return true
				}
				return false
			},
		),
	)
	monResourcesPredicates := builder.WithPredicates(
		predicate.NewPredicateFuncs(
			func(client client.Object) bool {
				labels := client.GetLabels()
				return labels == nil || labels[monLabelKey] != monLabelValue
			},
		),
	)
	monStatefulSetPredicates := builder.WithPredicates(
		predicate.NewPredicateFuncs(
			func(client client.Object) bool {
				name := client.GetName()
				return name == fmt.Sprintf("prometheus-%s", prometheusName) ||
					name == fmt.Sprintf("alertmanager-%s", alertmanagerName)
			},
		),
	)
	prometheusRulesPredicates := builder.WithPredicates(
		predicate.NewPredicateFuncs(
			func(client client.Object) bool {
				labels := client.GetLabels()
				return labels == nil || labels[monLabelKey] != monLabelValue
			},
		),
	)
	csvPredicates := builder.WithPredicates(
		predicate.NewPredicateFuncs(
			func(client client.Object) bool {
				return strings.HasPrefix(client.GetName(), ocsOperatorName)
			},
		),
	)
	enqueueManangedOCSRequest := handler.EnqueueRequestsFromMapFunc(
		func(client client.Object) []reconcile.Request {
			return []reconcile.Request{{
				NamespacedName: types.NamespacedName{
					Name:      managedOCSName,
					Namespace: client.GetNamespace(),
				},
			}}
		},
	)

	subPredicate := builder.WithPredicates(
		predicate.NewPredicateFuncs(
			func(client client.Object) bool {
				if instance, ok := client.(*opv1a1.Subscription); ok {
					// ignore if not a odf-operator subscription
					return instance.Spec.Package != odfOLMPackageName
				}
				return false
			},
		),
	)

	managedOCSController := ctrl.NewControllerManagedBy(mgr).
		WithOptions(*ctrlOptions).
		For(&v1.ManagedOCS{}, managedOCSPredicates).

		// Watch owned resources
		Owns(&ocsv1.StorageCluster{}).
		Owns(&promv1.Prometheus{}).
		Owns(&promv1.Alertmanager{}).
		Owns(&promv1a1.AlertmanagerConfig{}).
		Owns(&promv1.PrometheusRule{}).
		Owns(&promv1.ServiceMonitor{}).
		Owns(&netv1.NetworkPolicy{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).

		// Watch non-owned resources
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			enqueueManangedOCSRequest,
			secretPredicates,
		).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			enqueueManangedOCSRequest,
			configMapPredicates,
		).
		Watches(
			&source.Kind{Type: &promv1.PodMonitor{}},
			enqueueManangedOCSRequest,
			monResourcesPredicates,
		).
		Watches(
			&source.Kind{Type: &promv1.ServiceMonitor{}},
			enqueueManangedOCSRequest,
			monResourcesPredicates,
		).
		Watches(
			&source.Kind{Type: &promv1.PrometheusRule{}},
			enqueueManangedOCSRequest,
			prometheusRulesPredicates,
		).
		Watches(
			&source.Kind{Type: &appsv1.StatefulSet{}},
			enqueueManangedOCSRequest,
			monStatefulSetPredicates,
		).
		Watches(
			&source.Kind{Type: &ocsv1.OCSInitialization{}},
			enqueueManangedOCSRequest,
		).
		Watches(
			&source.Kind{Type: &opv1a1.ClusterServiceVersion{}},
			enqueueManangedOCSRequest,
			csvPredicates,
		).
		Watches(
			&source.Kind{Type: &opv1a1.Subscription{}},
			enqueueManangedOCSRequest,
			subPredicate,
		)

	if r.AvailableCRDs[EgressFirewallCRD] {
		managedOCSController = managedOCSController.Owns(&ovnv1.EgressFirewall{})
	}
	if r.AvailableCRDs[EgressNetworkPolicyCRD] {
		managedOCSController = managedOCSController.Owns(&openshiftv1.EgressNetworkPolicy{})
	}

	// Create the controller
	return managedOCSController.Complete(r)
}

// Reconcile changes to all owned resource based on the infromation provided by the ManagedOCS resource
func (r *ManagedOCSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("req.Namespace", req.Namespace, "req.Name", req.Name)
	log.Info("Starting reconcile for ManagedOCS")

	// Initalize the reconciler properties from the request
	r.initReconciler(ctx, req)

	// Load the managed ocs resource (input)
	if err := r.get(r.managedOCS); err != nil {
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
		statusErr = r.Client.Status().Update(r.ctx, r.managedOCS)
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

func (r *ManagedOCSReconciler) initReconciler(ctx context.Context, req ctrl.Request) {
	r.ctx = ctx
	r.namespace = req.NamespacedName.Namespace
	r.addonParams = make(map[string]string)
	r.DeploymentType = strings.ToLower(r.DeploymentType)

	r.managedOCS = &v1.ManagedOCS{}
	r.managedOCS.Name = req.NamespacedName.Name
	r.managedOCS.Namespace = r.namespace

	r.storageCluster = &ocsv1.StorageCluster{}
	r.storageCluster.Name = storageClusterName
	r.storageCluster.Namespace = r.namespace

	r.egressNetworkPolicy = &openshiftv1.EgressNetworkPolicy{}
	r.egressNetworkPolicy.Name = egressNetworkPolicyName
	r.egressNetworkPolicy.Namespace = r.namespace

	r.egressFirewall = &ovnv1.EgressFirewall{}
	r.egressFirewall.Name = egressFirewallName
	r.egressFirewall.Namespace = r.namespace

	r.ingressNetworkPolicy = &netv1.NetworkPolicy{}
	r.ingressNetworkPolicy.Name = ingressNetworkPolicyName
	r.ingressNetworkPolicy.Namespace = r.namespace

	r.cephIngressNetworkPolicy = &netv1.NetworkPolicy{}
	r.cephIngressNetworkPolicy.Name = cephIngressNetworkPolicyName
	r.cephIngressNetworkPolicy.Namespace = r.namespace

	r.providerAPIServerNetworkPolicy = &netv1.NetworkPolicy{}
	r.providerAPIServerNetworkPolicy.Name = providerApiServerNetworkPolicyName
	r.providerAPIServerNetworkPolicy.Namespace = r.namespace

	r.prometheusProxyNetworkPolicy = &netv1.NetworkPolicy{}
	r.prometheusProxyNetworkPolicy.Name = prometheusProxyNetworkPolicyName
	r.prometheusProxyNetworkPolicy.Namespace = r.namespace

	r.prometheus = &promv1.Prometheus{}
	r.prometheus.Name = prometheusName
	r.prometheus.Namespace = r.namespace

	r.prometheusService = &corev1.Service{}
	r.prometheusService.Name = prometheusServiceName
	r.prometheusService.Namespace = r.namespace

	r.dmsRule = &promv1.PrometheusRule{}
	r.dmsRule.Name = dmsRuleName
	r.dmsRule.Namespace = r.namespace

	r.alertmanager = &promv1.Alertmanager{}
	r.alertmanager.Name = alertmanagerName
	r.alertmanager.Namespace = r.namespace

	r.pagerdutySecret = &corev1.Secret{}
	r.pagerdutySecret.Name = r.PagerdutySecretName
	r.pagerdutySecret.Namespace = r.namespace

	r.smtpSecret = &corev1.Secret{}
	r.smtpSecret.Name = r.SMTPSecretName
	r.smtpSecret.Namespace = r.namespace

	r.deadMansSnitchSecret = &corev1.Secret{}
	r.deadMansSnitchSecret.Name = r.DeadMansSnitchSecretName
	r.deadMansSnitchSecret.Namespace = r.namespace

	r.alertmanagerConfig = &promv1a1.AlertmanagerConfig{}
	r.alertmanagerConfig.Name = alertmanagerConfigName
	r.alertmanagerConfig.Namespace = r.namespace

	r.k8sMetricsServiceMonitor = &promv1.ServiceMonitor{}
	r.k8sMetricsServiceMonitor.Name = k8sMetricsServiceMonitorName
	r.k8sMetricsServiceMonitor.Namespace = r.namespace

	r.alertRelabelConfigSecret = &corev1.Secret{}
	r.alertRelabelConfigSecret.Name = alertRelabelConfigSecretName
	r.alertRelabelConfigSecret.Namespace = r.namespace

	r.onboardingValidationKeySecret = &corev1.Secret{}
	r.onboardingValidationKeySecret.Name = onboardingValidationKeySecretName
	r.onboardingValidationKeySecret.Namespace = r.namespace

	r.prometheusKubeRBACConfigMap = &corev1.ConfigMap{}
	r.prometheusKubeRBACConfigMap.Name = templates.PrometheusKubeRBACPoxyConfigMapName
	r.prometheusKubeRBACConfigMap.Namespace = r.namespace

	r.rhobsRemoteWriteConfigSecret = &corev1.Secret{}
	r.rhobsRemoteWriteConfigSecret.Name = r.RHOBSSecretName
	r.rhobsRemoteWriteConfigSecret.Namespace = r.namespace

	r.csiKMSConnectionDetailsConfigMap = &corev1.ConfigMap{}
	r.csiKMSConnectionDetailsConfigMap.Name = csiKMSConnectionDetailsConfigMapName
	r.csiKMSConnectionDetailsConfigMap.Namespace = r.namespace
}

func (r *ManagedOCSReconciler) reconcilePhases() (reconcile.Result, error) {
	// Uninstallation depends on the status of the components.
	// We are checking the uninstallation condition before getting the component status
	// to mitigate scenarios where changes to the component status occurs while the uninstallation logic is running.
	initiateUninstall := r.checkUninstallCondition()
	// Update the status of the components
	r.updateComponentStatus()

	if !r.managedOCS.DeletionTimestamp.IsZero() {
		if r.verifyComponentsDoNotExist() {
			// Deleting OCS CSV from the namespace
			r.Log.Info("deleting OCS CSV")
			if err := r.deleteCSVByPrefix(ocsOperatorName); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to delete csv: %v", err)
			}

			r.Log.Info("removing finalizer from the ManagedOCS resource")
			r.managedOCS.SetFinalizers(utils.Remove(r.managedOCS.GetFinalizers(), ManagedOCSFinalizer))
			if err := r.Client.Update(r.ctx, r.managedOCS); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer from managedOCS: %v", err)
			}
			r.Log.Info("finallizer removed successfully")

		} else {
			// Storage cluster needs to be deleted before we delete the CSV so we can not leave it to the
			// k8s garbage collector to delete it
			r.Log.Info("deleting storagecluster")
			if err := r.delete(r.storageCluster); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to delete storagecluster: %v", err)
			}

			// Deleting all storage systems from the namespace
			r.Log.Info("deleting storageSystems")
			storageSystemList := odfv1a1.StorageSystemList{}
			if err := r.list(&storageSystemList); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to list storageSystem resource: %v", err)
			}
			for i := range storageSystemList.Items {
				storageSystem := storageSystemList.Items[i]
				if err := r.delete(&storageSystem); err != nil {
					return ctrl.Result{}, fmt.Errorf("unable to delete storageSystem: %v", err)
				}
			}
		}

	} else if r.managedOCS.UID != "" {
		if !utils.Contains(r.managedOCS.GetFinalizers(), ManagedOCSFinalizer) {
			r.Log.V(-1).Info("finalizer missing on the managedOCS resource, adding...")
			r.managedOCS.SetFinalizers(append(r.managedOCS.GetFinalizers(), ManagedOCSFinalizer))
			if err := r.update(r.managedOCS); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update managedOCS with finalizer: %v", err)
			}
		}

		// Find the effective reconcile strategy
		r.reconcileStrategy = v1.ReconcileStrategyStrict
		if strings.EqualFold(string(r.managedOCS.Spec.ReconcileStrategy), string(v1.ReconcileStrategyNone)) {
			r.reconcileStrategy = v1.ReconcileStrategyNone
		}

		// Read the add-on parameters secret and store it an addonParams map
		addonParamSecret := &corev1.Secret{}
		addonParamSecret.Name = r.AddonParamSecretName
		addonParamSecret.Namespace = r.namespace
		if err := r.get(addonParamSecret); err != nil {
			return ctrl.Result{}, fmt.Errorf("Failed to get the addon parameters secret %v", r.AddonParamSecretName)
		}
		for key, value := range addonParamSecret.Data {
			r.addonParams[key] = string(value)
		}

		// Deprecating consumer requested capacity addon parameter by overriding it with a very high pre-defined value
		if r.DeploymentType == consumerDeploymentType {
			r.Log.Info(fmt.Sprintf("Overriding addon parameter '%s' (currently '%s') with 1024", storageSizeKey, r.addonParams[storageSizeKey]))
			r.addonParams[storageSizeKey] = "1024"

			r.Log.Info(fmt.Sprintf("Overriding addon parameter '%s' (currently '%s') with Ti", storageUnitKey, r.addonParams[storageUnitKey]))
			r.addonParams[storageUnitKey] = "Ti"
		}

		// Reconcile the different resources
		if err := r.reconcileODFSubscription(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileRookCephOperatorConfig(); err != nil {
			return ctrl.Result{}, err
		}
		// Reconciles only for provider deployment type
		if err := r.reconcileOnboardingValidationSecret(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileStorageCluster(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileCSV(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileCSIKMSConnectionDetailsConfigMap(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileAlertRelabelConfigSecret(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcilePrometheusKubeRBACConfigMap(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcilePrometheusService(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcilePrometheus(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileAlertmanager(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileAlertmanagerConfig(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileK8SMetricsServiceMonitor(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileMonitoringResources(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileDMSPrometheusRule(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileOCSInitialization(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcilePrometheusProxyNetworkPolicy(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileEgressFirewall(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileEgressNetworkPolicy(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileIngressNetworkPolicy(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileCephIngressNetworkPolicy(); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileProviderAPIServerNetworkPolicy(); err != nil {
			return ctrl.Result{}, err
		}

		r.managedOCS.Status.ReconcileStrategy = r.reconcileStrategy

		// Check if we need and can uninstall
		if initiateUninstall && r.areComponentsReadyForUninstall() {
			switch r.DeploymentType {
			case convergedDeploymentType, consumerDeploymentType:
				found, err := r.hasOCSVolumes()
				if err != nil {
					return ctrl.Result{}, err
				}
				if found {
					r.Log.Info("Found consumer PVs using OCS storageclasses, cannot proceed with uninstallation")
					return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
				}
			case providerDeploymentType:
				found, err := r.hasOCSStorageConsumers()
				if err != nil {
					return ctrl.Result{}, err
				}
				if found {
					r.Log.Info("Found OCS storage consumers, cannot proceed with uninstallation")
					return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
				}
			}
			r.Log.Info("starting OCS uninstallation - deleting managedocs")
			if err := r.delete(r.managedOCS); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to delete managedocs: %v", err)
			}
			// Refreshing local managedOCS object after deletion is scheduled
			// to avoid conflict while updating status
			if err := r.get(r.managedOCS); err != nil {
				if !errors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
				r.Log.V(-1).Info("Trying to reload ManagedOCS resource after delete failed, ManagedOCS resource not found")
			}
		}

	} else if initiateUninstall {
		return ctrl.Result{}, r.removeOLMComponents()
	}

	return ctrl.Result{}, nil
}

func (r *ManagedOCSReconciler) updateComponentStatus() {
	// Getting the status of the StorageCluster component.
	scStatus := &r.managedOCS.Status.Components.StorageCluster
	if err := r.get(r.storageCluster); err == nil {
		if r.storageCluster.Status.Phase == "Ready" {
			scStatus.State = v1.ComponentReady
		} else {
			scStatus.State = v1.ComponentPending
		}
	} else if errors.IsNotFound(err) {
		scStatus.State = v1.ComponentNotFound
	} else {
		r.Log.V(-1).Info("error getting StorageCluster, setting compoment status to Unknown")
		scStatus.State = v1.ComponentUnknown
	}

	// Getting the status of the Prometheus component.
	promStatus := &r.managedOCS.Status.Components.Prometheus
	if err := r.get(r.prometheus); err == nil {
		promStatefulSet := &appsv1.StatefulSet{}
		promStatefulSet.Namespace = r.namespace
		promStatefulSet.Name = fmt.Sprintf("prometheus-%s", prometheusName)
		if err := r.get(promStatefulSet); err == nil {
			desiredReplicas := int32(1)
			if r.prometheus.Spec.Replicas != nil {
				desiredReplicas = *r.prometheus.Spec.Replicas
			}
			if promStatefulSet.Status.ReadyReplicas != desiredReplicas {
				promStatus.State = v1.ComponentPending
			} else {
				promStatus.State = v1.ComponentReady
			}
		} else {
			promStatus.State = v1.ComponentPending
		}
	} else if errors.IsNotFound(err) {
		promStatus.State = v1.ComponentNotFound
	} else {
		r.Log.V(-1).Info("error getting Prometheus, setting compoment status to Unknown")
		promStatus.State = v1.ComponentUnknown
	}

	// Getting the status of the Alertmanager component.
	amStatus := &r.managedOCS.Status.Components.Alertmanager
	if err := r.get(r.alertmanager); err == nil {
		amStatefulSet := &appsv1.StatefulSet{}
		amStatefulSet.Namespace = r.namespace
		amStatefulSet.Name = fmt.Sprintf("alertmanager-%s", alertmanagerName)
		if err := r.get(amStatefulSet); err == nil {
			desiredReplicas := int32(1)
			if r.alertmanager.Spec.Replicas != nil {
				desiredReplicas = *r.alertmanager.Spec.Replicas
			}
			if amStatefulSet.Status.ReadyReplicas != desiredReplicas {
				amStatus.State = v1.ComponentPending
			} else {
				amStatus.State = v1.ComponentReady
			}
		} else {
			amStatus.State = v1.ComponentPending
		}
	} else if errors.IsNotFound(err) {
		amStatus.State = v1.ComponentNotFound
	} else {
		r.Log.V(-1).Info("error getting Alertmanager, setting compoment status to Unknown")
		amStatus.State = v1.ComponentUnknown
	}
}

func (r *ManagedOCSReconciler) verifyComponentsDoNotExist() bool {
	subComponent := r.managedOCS.Status.Components

	return subComponent.StorageCluster.State == v1.ComponentNotFound
}

func (r *ManagedOCSReconciler) reconcileStorageCluster() error {
	r.Log.Info("Reconciling StorageCluster")

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.storageCluster, func() error {
		if err := r.own(r.storageCluster); err != nil {
			return err
		}

		// Handle only strict mode reconciliation
		if r.reconcileStrategy == v1.ReconcileStrategyStrict {
			var desired *ocsv1.StorageCluster = nil
			switch r.DeploymentType {
			case convergedDeploymentType:
				var err error
				if desired, err = r.getDesiredConvergedStorageCluster(); err != nil {
					return err
				}
			case consumerDeploymentType:
				var err error
				if desired, err = r.getDesiredConsumerStorageCluster(); err != nil {
					return err
				}
			case providerDeploymentType:
				var err error
				if desired, err = r.getDesiredProviderStorageCluster(); err != nil {
					return err
				}
			default:
				return fmt.Errorf("Invalid deployment type value: %v", r.DeploymentType)
			}
			// Override storage cluster spec with desired spec from the template.
			// We do not replace meta or status on purpose
			r.storageCluster.Spec = desired.Spec
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *ManagedOCSReconciler) getDesiredConvergedStorageCluster() (*ocsv1.StorageCluster, error) {
	sizeAsString := r.addonParams[storageSizeKey]

	// Setting hardcoded value here to force no MCG deployment
	enableMCGAsString := "false"
	if enableMCGRaw, exists := r.addonParams[enableMCGKey]; exists {
		enableMCGAsString = enableMCGRaw
	}

	r.Log.Info("Requested add-on settings", storageSizeKey, sizeAsString, enableMCGKey, enableMCGAsString)
	desiredDeviceSetCount, err := strconv.Atoi(sizeAsString)
	if err != nil {
		return nil, fmt.Errorf("Invalid storage cluster size value: %v", sizeAsString)
	}

	// Get the storage device set count of the current storage cluster
	currDeviceSetCount := 0
	if desiredStorageDeviceSet := findStorageDeviceSet(r.storageCluster.Spec.StorageDeviceSets, deviceSetName); desiredStorageDeviceSet != nil {
		currDeviceSetCount = desiredStorageDeviceSet.Count
	}

	// Get the desired storage device set from storage cluster template
	sc := templates.ConvergedStorageClusterTemplate.DeepCopy()
	var ds *ocsv1.StorageDeviceSet = nil
	if desiredStorageDeviceSet := findStorageDeviceSet(sc.Spec.StorageDeviceSets, deviceSetName); desiredStorageDeviceSet != nil {
		ds = desiredStorageDeviceSet
	}

	// Prevent downscaling by comparing count from secret and count from storage cluster
	r.setDeviceSetCount(ds, desiredDeviceSetCount, currDeviceSetCount)

	// Check and enable MCG in Storage Cluster spec
	mcgEnable, err := strconv.ParseBool(enableMCGAsString)
	if err != nil {
		return nil, fmt.Errorf("Invalid Enable MCG value: %v", enableMCGAsString)
	}
	if err := r.ensureMCGDeployment(sc, mcgEnable); err != nil {
		return nil, err
	}

	return sc, nil
}

func (r *ManagedOCSReconciler) getDesiredProviderStorageCluster() (*ocsv1.StorageCluster, error) {
	sizeAsString := r.addonParams[storageSizeKey]

	// Setting hardcoded value here to force no MCG deployment
	enableMCGAsString := "false"
	if enableMCGRaw, exists := r.addonParams[enableMCGKey]; exists {
		enableMCGAsString = enableMCGRaw
	}
	r.Log.Info("Requested add-on settings", storageSizeKey, sizeAsString, enableMCGKey, enableMCGAsString)
	desiredSize, err := strconv.Atoi(sizeAsString)
	if err != nil {
		return nil, fmt.Errorf("Invalid storage cluster size value: %v", sizeAsString)
	}

	// Convert the desired size to the device set count based on the underlaying OSD size
	desiredDeviceSetCount := int(math.Ceil(float64(desiredSize) / templates.ProviderOSDSizeInTiB))

	// Get the storage device set count of the current storage cluster
	currDeviceSetCount := 0
	if desiredStorageDeviceSet := findStorageDeviceSet(r.storageCluster.Spec.StorageDeviceSets, deviceSetName); desiredStorageDeviceSet != nil {
		currDeviceSetCount = desiredStorageDeviceSet.Count
	}

	// Get the desired storage device set from storage cluster template
	sc := templates.ProviderStorageClusterTemplate.DeepCopy()
	var ds *ocsv1.StorageDeviceSet = nil
	if desiredStorageDeviceSet := findStorageDeviceSet(sc.Spec.StorageDeviceSets, deviceSetName); desiredStorageDeviceSet != nil {
		ds = desiredStorageDeviceSet
	}

	// Prevent downscaling by comparing count from secret and count from storage cluster
	r.setDeviceSetCount(ds, desiredDeviceSetCount, currDeviceSetCount)

	// Check and enable MCG in Storage Cluster spec
	mcgEnable, err := strconv.ParseBool(enableMCGAsString)
	if err != nil {
		return nil, fmt.Errorf("Invalid Enable MCG value: %v", enableMCGAsString)
	}
	if err := r.ensureMCGDeployment(sc, mcgEnable); err != nil {
		return nil, err
	}

	return sc, nil
}

func (r *ManagedOCSReconciler) getDesiredConsumerStorageCluster() (*ocsv1.StorageCluster, error) {
	storageSize := r.addonParams[storageSizeKey]
	storageUnit := r.addonParams[storageUnitKey]
	onboardingTicket := r.addonParams[onboardingTicketKey]
	storageProviderEndpoint := r.addonParams[storageProviderEndpointKey]

	// Setting hardcoded value here to force no MCG deployment
	enableMCGAsString := "false"
	if enableMCGRaw, exists := r.addonParams[enableMCGKey]; exists {
		enableMCGAsString = enableMCGRaw
	}

	r.Log.Info("Requested add-on settings", storageSizeKey, storageSize, storageUnitKey, storageUnit, enableMCGKey, enableMCGAsString,
		onboardingTicketKey, onboardingTicket, storageProviderEndpointKey, storageProviderEndpoint)

	sizeAsString := storageSize + storageUnit
	requestedCapacity, err := resource.ParseQuantity(sizeAsString)
	// Check if the requested capacity is valid
	if err != nil || requestedCapacity.Sign() == -1 {
		return nil, fmt.Errorf("Invalid storage cluster capacity value: %v", sizeAsString)
	}

	sc := templates.ConsumerStorageClusterTemplate.DeepCopy()
	externalStorageSpec := &sc.Spec.ExternalStorage
	externalStorageSpec.OnboardingTicket = onboardingTicket
	externalStorageSpec.StorageProviderEndpoint = storageProviderEndpoint
	externalStorageSpec.RequestedCapacity = &requestedCapacity

	// Check and enable MCG in Storage Cluster spec
	mcgEnable, err := strconv.ParseBool(enableMCGAsString)
	if err != nil {
		return nil, fmt.Errorf("Invalid Enable MCG value: %v", enableMCGAsString)
	}
	if err := r.ensureMCGDeployment(sc, mcgEnable); err != nil {
		return nil, err
	}

	return sc, nil
}

func (r *ManagedOCSReconciler) reconcileOnboardingValidationSecret() error {
	if r.DeploymentType != providerDeploymentType {
		r.Log.Info("Non provider deployment, skipping reconcile for onboarding validation secret")
		return nil
	}
	r.Log.Info("Reconciling onboardingValidationKeySecret")
	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.onboardingValidationKeySecret, func() error {
		if err := r.own(r.onboardingValidationKeySecret); err != nil {
			return err
		}
		onboardingValidationData := fmt.Sprintf(
			"-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----",
			strings.TrimSpace(r.addonParams[onboardingValidationKey]),
		)
		r.onboardingValidationKeySecret.Data = map[string][]byte{
			"key": []byte(onboardingValidationData),
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to update onboardingValidationKeySecret: %v", err)
	}
	return nil
}

func (r *ManagedOCSReconciler) reconcileCSIKMSConnectionDetailsConfigMap() error {
	if r.DeploymentType != consumerDeploymentType {
		r.Log.Info(fmt.Sprintf("Non consumer deployment, skipping reconcile for %s config map", csiKMSConnectionDetailsConfigMapName))
		return nil
	}
	r.Log.Info(fmt.Sprintf("Reconciling %s configMap", csiKMSConnectionDetailsConfigMapName))

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.csiKMSConnectionDetailsConfigMap, func() error {
		if err := r.own(r.csiKMSConnectionDetailsConfigMap); err != nil {
			return err
		}

		awsStsMetadataTestAsString := string(
			utils.ToJsonOrDie(
				map[string]string{
					"encryptionKMSType": "aws-sts-metadata",
					"secretName":        "ceph-csi-aws-credentials",
				},
			),
		)
		r.csiKMSConnectionDetailsConfigMap.Data = map[string]string{
			"aws-sts-metadata-test": awsStsMetadataTestAsString,
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to create or update %s config map: %v", csiKMSConnectionDetailsConfigMapName, err)
	}

	return nil
}

// AlertRelabelConfigSecret will have configuration for relabeling the alerts that are firing.
// It will add namespace label to firing alerts before they are sent to the alertmanager
func (r *ManagedOCSReconciler) reconcileAlertRelabelConfigSecret() error {
	r.Log.Info("Reconciling alertRelabelConfigSecret")

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.alertRelabelConfigSecret, func() error {
		if err := r.own(r.alertRelabelConfigSecret); err != nil {
			return err
		}

		alertRelabelConfig := []struct {
			SourceLabels []string `yaml:"source_labels"`
			TargetLabel  string   `yaml:"target_label,omitempty"`
			Replacement  string   `yaml:"replacement,omitempty"`
		}{
			{
				SourceLabels: []string{"namespace"},
				TargetLabel:  "alertnamespace",
			},
			{
				TargetLabel: "namespace",
				Replacement: r.namespace,
			},
		}

		config, err := yaml.Marshal(alertRelabelConfig)
		if err != nil {
			return fmt.Errorf("Unable to encode alert relabel conifg: %v", err)
		}

		r.alertRelabelConfigSecret.Data = map[string][]byte{
			alertRelabelConfigSecretKey: config,
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("Unable to create/update AlertRelabelConfigSecret: %v", err)
	}

	return nil
}

func (r *ManagedOCSReconciler) reconcilePrometheusKubeRBACConfigMap() error {
	r.Log.Info("Reconciling kubeRBACConfigMap")

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.prometheusKubeRBACConfigMap, func() error {
		if err := r.own(r.prometheusKubeRBACConfigMap); err != nil {
			return err
		}

		r.prometheusKubeRBACConfigMap.Data = templates.KubeRBACProxyConfigMap.DeepCopy().Data

		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to create kubeRBACConfig config map: %v", err)
	}

	return nil
}

// reconcilePrometheusService function wait for prometheus Service
// to start and sets appropriate annotation for 'service-ca' controller
func (r *ManagedOCSReconciler) reconcilePrometheusService() error {
	r.Log.Info("Reconciling PrometheusService")

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.prometheusService, func() error {
		if err := r.own(r.prometheusService); err != nil {
			return err
		}

		r.prometheusService.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "https",
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(templates.KubeRBACProxyPortNumber),
				TargetPort: intstr.FromString("https"),
			},
		}
		r.prometheusService.Spec.Selector = map[string]string{
			"app.kubernetes.io/name": r.prometheusService.Name,
		}
		utils.AddAnnotation(
			r.prometheusService,
			"service.beta.openshift.io/serving-cert-secret-name",
			templates.PrometheusServingCertSecretName,
		)
		utils.AddAnnotation(
			r.prometheusService,
			"service.alpha.openshift.io/serving-cert-secret-name",
			templates.PrometheusServingCertSecretName,
		)
		// This label is required to enable us to use metrics federation
		// mechanism provided by Managed-tenants
		utils.AddLabel(r.prometheusService, monLabelKey, monLabelValue)
		return nil
	})
	return err
}

func (r *ManagedOCSReconciler) reconcilePrometheus() error {
	r.Log.Info("Reconciling Prometheus")

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.prometheus, func() error {
		if err := r.own(r.prometheus); err != nil {
			return err
		}

		desired := templates.PrometheusTemplate.DeepCopy()
		utils.AddLabel(r.prometheus, monLabelKey, monLabelValue)

		// use the container image of kube-rbac-proxy that comes in deployer CSV
		// for prometheus kube-rbac-proxy sidecar
		deployerCSV, err := r.getCSVByPrefix(deployerCSVPrefix)
		if err != nil {
			return fmt.Errorf("Unable to set image for kube-rbac-proxy container: %v", err)
		}

		deployerCSVDeployments := deployerCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
		var deployerCSVDeployment *opv1a1.StrategyDeploymentSpec = nil
		for key := range deployerCSVDeployments {
			deployment := &deployerCSVDeployments[key]
			if deployment.Name == "ocs-osd-controller-manager" {
				deployerCSVDeployment = deployment
			}
		}

		deployerCSVContainers := deployerCSVDeployment.Spec.Template.Spec.Containers
		var kubeRbacImage string
		for key := range deployerCSVContainers {
			container := deployerCSVContainers[key]
			if container.Name == "kube-rbac-proxy" {
				kubeRbacImage = container.Image
			}
		}

		prometheusContainers := desired.Spec.Containers
		for key := range prometheusContainers {
			container := &prometheusContainers[key]
			if container.Name == "kube-rbac-proxy" {
				container.Image = kubeRbacImage
			}
		}

		clusterVersion := &configv1.ClusterVersion{}
		clusterVersion.Name = "version"
		if err := r.get(clusterVersion); err != nil {
			return err
		}

		r.prometheus.Spec = desired.Spec
		r.prometheus.Spec.ExternalLabels["clusterId"] = string(clusterVersion.Spec.ClusterID)
		r.prometheus.Spec.Alerting.Alertmanagers[0].Namespace = r.namespace
		r.prometheus.Spec.AdditionalAlertRelabelConfigs = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: alertRelabelConfigSecretName,
			},
			Key: alertRelabelConfigSecretKey,
		}
		if r.DeploymentType != convergedDeploymentType {
			if err := r.get(r.rhobsRemoteWriteConfigSecret); err != nil && !errors.IsNotFound(err) {
				return err
			}
			if r.rhobsRemoteWriteConfigSecret.UID != "" {
				rhobsSecretData := r.rhobsRemoteWriteConfigSecret.Data
				if _, found := rhobsSecretData[rhobsRemoteWriteConfigIdSecretKey]; !found {
					return fmt.Errorf("rhobs secret does not contain a value for key %v", rhobsRemoteWriteConfigIdSecretKey)
				}
				if _, found := rhobsSecretData[rhobsRemoteWriteConfigSecretName]; !found {
					return fmt.Errorf("rhobs secret does not contain a value for key %v", rhobsRemoteWriteConfigSecretName)
				}
				rhobsAudience, found := rhobsSecretData["rhobs-audience"]
				if !found {
					return fmt.Errorf("rhobs secret does not contain a value for key rhobs-audience")
				}
				remoteWriteSpec := &r.prometheus.Spec.RemoteWrite[0]
				remoteWriteSpec.URL = r.RHOBSEndpoint
				remoteWriteSpec.OAuth2.ClientID.Secret.LocalObjectReference.Name = r.RHOBSSecretName
				remoteWriteSpec.OAuth2.ClientID.Secret.Key = rhobsRemoteWriteConfigIdSecretKey
				remoteWriteSpec.OAuth2.ClientSecret.LocalObjectReference.Name = r.RHOBSSecretName
				remoteWriteSpec.OAuth2.ClientSecret.Key = rhobsRemoteWriteConfigSecretName
				remoteWriteSpec.OAuth2.TokenURL = r.RHSSOTokenEndpoint
				remoteWriteSpec.OAuth2.EndpointParams["audience"] = string(rhobsAudience)
			} else {
				r.Log.Info("RHOBS remote write config secret not found , disabling remote write")
				r.prometheus.Spec.RemoteWrite = nil
			}
		} else {
			// removing remote write spec for converged clusters
			r.prometheus.Spec.RemoteWrite = nil
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to update Prometheus: %v", err)
	}

	return nil
}

func (r *ManagedOCSReconciler) reconcileDMSPrometheusRule() error {
	r.Log.Info("Reconciling DMS Prometheus Rule")

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.dmsRule, func() error {
		if err := r.own(r.dmsRule); err != nil {
			return err
		}

		desired := templates.DMSPrometheusRuleTemplate.DeepCopy()

		for _, group := range desired.Spec.Groups {
			if group.Name == "snitch-alert" {
				for _, rule := range group.Rules {
					if rule.Alert == "DeadMansSnitch" {
						rule.Labels["namespace"] = r.namespace
					}
				}
			}
		}

		r.dmsRule.Spec = desired.Spec

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *ManagedOCSReconciler) reconcileOCSInitialization() error {
	r.Log.Info("Reconciling OCSInitialization")

	ocsInitList := ocsv1.OCSInitializationList{}
	if err := r.list(&ocsInitList); err != nil {
		return fmt.Errorf("Could to list OCSInitialization resources: %v", err)
	}
	if len(ocsInitList.Items) == 0 {
		r.Log.V(-1).Info("OCSInitialization resource not found")
	} else {
		obj := &ocsInitList.Items[0]
		if !obj.Spec.EnableCephTools {
			obj.Spec.EnableCephTools = true
			if err := r.update(obj); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ManagedOCSReconciler) reconcileAlertmanager() error {
	r.Log.Info("Reconciling Alertmanager")
	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.alertmanager, func() error {
		if err := r.own(r.alertmanager); err != nil {
			return err
		}

		desired := templates.AlertmanagerTemplate.DeepCopy()
		desired.Spec.AlertmanagerConfigSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				monLabelKey: monLabelValue,
			},
		}
		r.alertmanager.Spec = desired.Spec
		utils.AddLabel(r.alertmanager, monLabelKey, monLabelValue)

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *ManagedOCSReconciler) reconcileAlertmanagerConfig() error {
	r.Log.Info("Reconciling AlertmanagerConfig secret")

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.alertmanagerConfig, func() error {
		if err := r.own(r.alertmanagerConfig); err != nil {
			return err
		}

		if err := r.get(r.pagerdutySecret); err != nil {
			return fmt.Errorf("Unable to get pagerduty secret: %v", err)
		}
		pagerdutySecretData := r.pagerdutySecret.Data
		pagerdutyServiceKey := string(pagerdutySecretData["PAGERDUTY_KEY"])
		if pagerdutyServiceKey == "" {
			return fmt.Errorf("Pagerduty secret does not contain a PAGERDUTY_KEY entry")
		}

		if r.deadMansSnitchSecret.UID == "" {
			if err := r.get(r.deadMansSnitchSecret); err != nil {
				return fmt.Errorf("Unable to get DeadMan's Snitch secret: %v", err)
			}
		}
		dmsURL := string(r.deadMansSnitchSecret.Data["SNITCH_URL"])
		if dmsURL == "" {
			return fmt.Errorf("DeadMan's Snitch secret does not contain a SNITCH_URL entry")
		}

		alertingAddressList := []string{}
		i := 0
		for {
			alertingAddressKey := fmt.Sprintf("%s-%v", notificationEmailKeyPrefix, i)
			alertingAddress, found := r.addonParams[alertingAddressKey]
			i++
			if found {
				alertingAddressAsString := alertingAddress
				if alertingAddressAsString != "" {
					alertingAddressList = append(alertingAddressList, alertingAddressAsString)
				}
			} else {
				break
			}
		}

		if r.smtpSecret.UID == "" {
			if err := r.get(r.smtpSecret); err != nil {
				return fmt.Errorf("Unable to get SMTP secret: %v", err)
			}
		}
		smtpSecretData := r.smtpSecret.Data
		smtpHost := string(smtpSecretData["host"])
		if smtpHost == "" {
			return fmt.Errorf("smtp secret does not contain a host entry")
		}
		smtpPort := string(smtpSecretData["port"])
		if smtpPort == "" {
			return fmt.Errorf("smtp secret does not contain a port entry")
		}
		smtpUsername := string(smtpSecretData["username"])
		if smtpUsername == "" {
			return fmt.Errorf("smtp secret does not contain a username entry")
		}
		smtpPassword := string(smtpSecretData["password"])
		if smtpPassword == "" {
			return fmt.Errorf("smtp secret does not contain a password entry")
		}
		smtpHTML, err := os.ReadFile(r.CustomerNotificationHTMLPath)
		if err != nil {
			return fmt.Errorf("unable to read customernotification.html file: %v", err)
		}

		desired := templates.AlertmanagerConfigTemplate.DeepCopy()
		for i := range desired.Spec.Receivers {
			receiver := &desired.Spec.Receivers[i]
			switch receiver.Name {
			case "pagerduty":
				receiver.PagerDutyConfigs[0].ServiceKey.Key = "PAGERDUTY_KEY"
				receiver.PagerDutyConfigs[0].ServiceKey.LocalObjectReference.Name = r.PagerdutySecretName
				receiver.PagerDutyConfigs[0].Details[0].Key = "SOP"
				receiver.PagerDutyConfigs[0].Details[0].Value = r.SOPEndpoint
			case "DeadMansSnitch":
				receiver.WebhookConfigs[0].URL = &dmsURL
			case "SendGrid":
				if len(alertingAddressList) > 0 {
					receiver.EmailConfigs[0].Smarthost = fmt.Sprintf("%s:%s", smtpHost, smtpPort)
					receiver.EmailConfigs[0].AuthUsername = smtpUsername
					receiver.EmailConfigs[0].AuthPassword.LocalObjectReference.Name = r.SMTPSecretName
					receiver.EmailConfigs[0].AuthPassword.Key = "password"
					receiver.EmailConfigs[0].From = r.AlertSMTPFrom
					receiver.EmailConfigs[0].To = strings.Join(alertingAddressList, ", ")
					receiver.EmailConfigs[0].HTML = string(smtpHTML)
				} else {
					r.Log.V(-1).Info("Customer Email for alert notification is not provided")
					receiver.EmailConfigs = []promv1a1.EmailConfig{}
				}
			}
		}
		r.alertmanagerConfig.Spec = desired.Spec
		utils.AddLabel(r.alertmanagerConfig, monLabelKey, monLabelValue)

		return nil
	})

	return err
}

func (r *ManagedOCSReconciler) reconcileK8SMetricsServiceMonitor() error {
	r.Log.Info("Reconciling k8sMetricsServiceMonitor")

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.k8sMetricsServiceMonitor, func() error {
		if err := r.own(r.k8sMetricsServiceMonitor); err != nil {
			return err
		}
		desired := templates.K8sMetricsServiceMonitorTemplate.DeepCopy()
		r.k8sMetricsServiceMonitor.Spec = desired.Spec
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to update k8sMetricsServiceMonitor: %v", err)
	}
	return nil
}

// reconcileMonitoringResources labels all monitoring resources (ServiceMonitors, PodMonitors, and PrometheusRules)
// found in the target namespace with a label that matches the label selector the defined on the Prometheus resource
// we are reconciling in reconcilePrometheus. Doing so instructs the Prometheus instance to notice and react to these labeled
// monitoring resources
func (r *ManagedOCSReconciler) reconcileMonitoringResources() error {
	r.Log.Info("reconciling monitoring resources")

	podMonitorList := promv1.PodMonitorList{}
	if err := r.list(&podMonitorList); err != nil {
		return fmt.Errorf("Could not list pod monitors: %v", err)
	}
	for i := range podMonitorList.Items {
		obj := podMonitorList.Items[i]
		utils.AddLabel(obj, monLabelKey, monLabelValue)
		if err := r.update(obj); err != nil {
			return err
		}
	}

	serviceMonitorList := promv1.ServiceMonitorList{}
	if err := r.list(&serviceMonitorList); err != nil {
		return fmt.Errorf("Could not list service monitors: %v", err)
	}
	for i := range serviceMonitorList.Items {
		obj := serviceMonitorList.Items[i]
		utils.AddLabel(obj, monLabelKey, monLabelValue)
		if err := r.update(obj); err != nil {
			return err
		}
	}

	promRuleList := promv1.PrometheusRuleList{}
	if err := r.list(&promRuleList); err != nil {
		return fmt.Errorf("Could not list prometheus rules: %v", err)
	}
	for i := range promRuleList.Items {
		obj := promRuleList.Items[i]
		utils.AddLabel(obj, monLabelKey, monLabelValue)
		if err := r.update(obj); err != nil {
			return err
		}
	}

	return nil
}

// reconcileRookCephOperatorConfig is used to set resource request and limits on csi containers
func (r *ManagedOCSReconciler) reconcileRookCephOperatorConfig() error {
	rookConfigMap := &corev1.ConfigMap{}
	rookConfigMap.Name = rookConfigMapName
	rookConfigMap.Namespace = r.namespace

	if err := r.get(rookConfigMap); err != nil {
		// Because resource limits will not be set, failure to get the Rook ConfigMap results in failure to reconcile.
		return fmt.Errorf("Failed to get Rook ConfigMap: %v", err)
	}

	cloneRookConfigMap := rookConfigMap.DeepCopy()
	if cloneRookConfigMap.Data == nil {
		cloneRookConfigMap.Data = map[string]string{}
	}

	if r.DeploymentType == providerDeploymentType {
		cloneRookConfigMap.Data["ROOK_CSI_ENABLE_CEPHFS"] = "false"
		cloneRookConfigMap.Data["ROOK_CSI_ENABLE_RBD"] = "false"
	} else {
		cloneRookConfigMap.Data["CSI_RBD_PROVISIONER_RESOURCE"] = utils.MarshalRookResourceRequirements(utils.RookResourceRequirementsList{
			utils.RookResourceRequirements{
				Name:     "csi-provisioner",
				Resource: utils.GetResourceRequirements("csi-provisioner"),
			},
			utils.RookResourceRequirements{
				Name:     "csi-resizer",
				Resource: utils.GetResourceRequirements("csi-resizer"),
			},
			utils.RookResourceRequirements{
				Name:     "csi-attacher",
				Resource: utils.GetResourceRequirements("csi-attacher"),
			},
			utils.RookResourceRequirements{
				Name:     "csi-snapshotter",
				Resource: utils.GetResourceRequirements("csi-snapshotter"),
			},
			utils.RookResourceRequirements{
				Name:     "csi-rbdplugin",
				Resource: utils.GetResourceRequirements("csi-rbdplugin"),
			},
			utils.RookResourceRequirements{
				Name:     "liveness-prometheus",
				Resource: utils.GetResourceRequirements("liveness-prometheus"),
			},
		})

		cloneRookConfigMap.Data["CSI_RBD_PLUGIN_RESOURCE"] = utils.MarshalRookResourceRequirements(utils.RookResourceRequirementsList{
			{
				Name:     "driver-registrar",
				Resource: utils.GetResourceRequirements("driver-registrar"),
			},
			{
				Name:     "csi-rbdplugin",
				Resource: utils.GetResourceRequirements("csi-rbdplugin"),
			},
			{
				Name:     "liveness-prometheus",
				Resource: utils.GetResourceRequirements("liveness-prometheus"),
			},
		})

		cloneRookConfigMap.Data["CSI_CEPHFS_PROVISIONER_RESOURCE"] = utils.MarshalRookResourceRequirements(utils.RookResourceRequirementsList{
			{
				Name:     "csi-provisioner",
				Resource: utils.GetResourceRequirements("csi-provisioner"),
			},
			{
				Name:     "csi-resizer",
				Resource: utils.GetResourceRequirements("csi-resizer"),
			},
			{
				Name:     "csi-attacher",
				Resource: utils.GetResourceRequirements("csi-attacher"),
			},
			{
				Name:     "csi-snapshotter",
				Resource: utils.GetResourceRequirements("csi-snapshotter"),
			},
			{
				Name:     "csi-cephfsplugin",
				Resource: utils.GetResourceRequirements("csi-cephfsplugin"),
			},
			{
				Name:     "liveness-prometheus",
				Resource: utils.GetResourceRequirements("liveness-prometheus"),
			},
		})

		cloneRookConfigMap.Data["CSI_CEPHFS_PLUGIN_RESOURCE"] = utils.MarshalRookResourceRequirements(utils.RookResourceRequirementsList{
			{
				Name:     "driver-registrar",
				Resource: utils.GetResourceRequirements("driver-registrar"),
			},
			{
				Name:     "csi-cephfsplugin",
				Resource: utils.GetResourceRequirements("csi-cephfsplugin"),
			},
			{
				Name:     "liveness-prometheus",
				Resource: utils.GetResourceRequirements("liveness-prometheus"),
			},
		})
	}

	if !equality.Semantic.DeepEqual(rookConfigMap, cloneRookConfigMap) {
		if err := r.update(cloneRookConfigMap); err != nil {
			return fmt.Errorf("Failed to update Rook ConfigMap: %v", err)
		}
	}

	return nil
}

func (r *ManagedOCSReconciler) reconcileEgressNetworkPolicy() error {
	if !r.AvailableCRDs[EgressNetworkPolicyCRD] {
		r.Log.Info("EgressNetworkPolicy CRD not present, skipping reconcile for egress network policy")
		return nil
	}
	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.egressNetworkPolicy, func() error {
		if err := r.own(r.egressNetworkPolicy); err != nil {
			return err
		}
		desired := templates.EgressNetworkPolicyTemplate.DeepCopy()

		if r.DeploymentType != convergedDeploymentType {
			// Get the aws config map
			awsConfigMap := &corev1.ConfigMap{}
			awsConfigMap.Name = utils.IMDSConfigMapName
			awsConfigMap.Namespace = r.namespace
			if err := r.get(awsConfigMap); err != nil {
				return fmt.Errorf("Unable to get AWS ConfigMap: %v", err)
			}

			// Get the machine cidr
			vpcCIDR, ok := awsConfigMap.Data[utils.CIDRKey]
			if !ok {
				return fmt.Errorf("Unable to determine machine CIDR from AWS ConfigMap")
			}
			// Allow egress traffic to that cidr range
			var vpcEgressRules []openshiftv1.EgressNetworkPolicyRule
			cidrList := strings.Split(vpcCIDR, ";")
			for _, cidr := range cidrList {
				if cidr == "" {
					continue
				}
				vpcEgressRules = append(vpcEgressRules, openshiftv1.EgressNetworkPolicyRule{
					Type: openshiftv1.EgressNetworkPolicyRuleAllow,
					To: openshiftv1.EgressNetworkPolicyPeer{
						CIDRSelector: cidr,
					},
				})
			}

			// Inserting the VPC Egress rule in front of all other egress rules.
			// The order or rules matter
			// https://docs.openshift.com/container-platform/4.10/networking/openshift_sdn/configuring-egress-firewall.html#policy-rule-order_openshift-sdn-egress-firewall
			// Inserting this rule in the front ensures it comes before the EgressNetworkPolicyRuleDeny rule.
			desired.Spec.Egress = append(
				vpcEgressRules,
				desired.Spec.Egress...,
			)
		}

		if r.DeploymentType == consumerDeploymentType {

			storageProviderEndpointURL := r.addonParams[storageProviderEndpointKey]
			storageProviderEndpointHost := strings.Split(storageProviderEndpointURL, ":")[0]

			if storageProviderEndpointHost == "" {
				return fmt.Errorf("storageProviderEndpoint does not contain a host entry")
			}

			if ip := net.ParseIP(storageProviderEndpointHost); ip == nil {
				loadBalancerEgressRule := openshiftv1.EgressNetworkPolicyRule{}
				loadBalancerEgressRule.To.DNSName = storageProviderEndpointHost
				loadBalancerEgressRule.Type = openshiftv1.EgressNetworkPolicyRuleAllow

				desired.Spec.Egress = append(
					[]openshiftv1.EgressNetworkPolicyRule{
						loadBalancerEgressRule,
					},
					desired.Spec.Egress...,
				)
			}
		}

		if r.deadMansSnitchSecret.UID == "" {
			if err := r.get(r.deadMansSnitchSecret); err != nil {
				return fmt.Errorf("Unable to get DMS secret: %v", err)
			}
		}
		snitchStr := string(r.deadMansSnitchSecret.Data["SNITCH_URL"])
		if snitchStr == "" {
			return fmt.Errorf("DMS secret does not contain a SNITCH_URL entry")
		}
		snitchURL, err := url.Parse(snitchStr)
		if err != nil {
			return fmt.Errorf("Unable to parse DMS url: %v", err)
		}
		if snitchURL.Hostname() == "" {
			return fmt.Errorf("DMS snitch url %q does not have a valid hostname", snitchURL)
		}

		if r.smtpSecret.UID == "" {
			if err := r.get(r.smtpSecret); err != nil {
				return fmt.Errorf("Unable to get SMTP secret: %v", err)
			}
		}
		smtpHost := string(r.smtpSecret.Data["host"])
		if smtpHost == "" {
			return fmt.Errorf("smtp secret does not contain a host entry")
		}

		rhobsURL, err := url.Parse(r.RHOBSEndpoint)
		if err != nil {
			return fmt.Errorf("Unable to parse RHOBS url: %v", err)
		}
		if rhobsURL.Hostname() == "" {
			return fmt.Errorf("RHOBS url %q does not have a valid hostname", rhobsURL)
		}

		ssoURL, err := url.Parse(r.RHSSOTokenEndpoint)
		if err != nil {
			return fmt.Errorf("Unable to parse RHSSOTokenEndpoint url: %v", err)
		}
		if ssoURL.Hostname() == "" {
			return fmt.Errorf("RHSSOTokenEndpoint url %q does not have a valid hostname", ssoURL)
		}

		dmsEgressRule := openshiftv1.EgressNetworkPolicyRule{}
		dmsEgressRule.To.DNSName = snitchURL.Hostname()
		dmsEgressRule.Type = openshiftv1.EgressNetworkPolicyRuleAllow

		smtpEgressRule := openshiftv1.EgressNetworkPolicyRule{}
		smtpEgressRule.To.DNSName = smtpHost
		smtpEgressRule.Type = openshiftv1.EgressNetworkPolicyRuleAllow

		rhobsEgressRule := openshiftv1.EgressNetworkPolicyRule{}
		rhobsEgressRule.To.DNSName = rhobsURL.Hostname()
		rhobsEgressRule.Type = openshiftv1.EgressNetworkPolicyRuleAllow

		ssoEgressRule := openshiftv1.EgressNetworkPolicyRule{}
		ssoEgressRule.To.DNSName = ssoURL.Hostname()
		ssoEgressRule.Type = openshiftv1.EgressNetworkPolicyRuleAllow

		desired.Spec.Egress = append(
			[]openshiftv1.EgressNetworkPolicyRule{
				dmsEgressRule,
				smtpEgressRule,
				rhobsEgressRule,
				ssoEgressRule,
			},
			desired.Spec.Egress...,
		)
		r.egressNetworkPolicy.Spec = desired.Spec
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to update egressNetworkPolicy: %v", err)
	}
	return nil
}

func (r *ManagedOCSReconciler) reconcileEgressFirewall() error {
	if !r.AvailableCRDs[EgressFirewallCRD] {
		r.Log.Info("EgressFirewall CRD not present, skipping reconcile for egress firewall")
		return nil
	}

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.egressFirewall, func() error {
		if err := r.own(r.egressFirewall); err != nil {
			return err
		}
		desired := templates.EgressFirewallTemplate.DeepCopy()

		if r.DeploymentType != convergedDeploymentType {
			// Get the aws config map
			awsConfigMap := &corev1.ConfigMap{}
			awsConfigMap.Name = utils.IMDSConfigMapName
			awsConfigMap.Namespace = r.namespace
			if err := r.get(awsConfigMap); err != nil {
				return fmt.Errorf("Unable to get AWS ConfigMap: %v", err)
			}

			// Get the machine cidr
			vpcCIDR, ok := awsConfigMap.Data[utils.CIDRKey]
			if !ok {
				return fmt.Errorf("Unable to determine machine CIDR from AWS ConfigMap")
			}
			// Allow egress traffic to that cidr range
			var vpcEgressRules []ovnv1.EgressFirewallRule
			cidrList := strings.Split(vpcCIDR, ";")
			for _, cidr := range cidrList {
				if cidr == "" {
					continue
				}
				vpcEgressRules = append(vpcEgressRules, ovnv1.EgressFirewallRule{
					Type: ovnv1.EgressFirewallRuleAllow,
					To: ovnv1.EgressFirewallDestination{
						CIDRSelector: cidr,
					},
				})
			}

			// Inserting the VPC Egress rule in front of all other egress rules.
			// The order or rules matter
			// https://docs.openshift.com/container-platform/4.10/networking/openshift_sdn/configuring-egress-firewall.html#policy-rule-order_openshift-sdn-egress-firewall
			// Inserting this rule in the front ensures it comes before the EgressNetworkPolicyRuleDeny rule.
			desired.Spec.Egress = append(
				vpcEgressRules,
				desired.Spec.Egress...,
			)
		}

		if r.DeploymentType == consumerDeploymentType {

			storageProviderEndpointURL := r.addonParams[storageProviderEndpointKey]
			storageProviderEndpointHost := strings.Split(storageProviderEndpointURL, ":")[0]

			if storageProviderEndpointHost == "" {
				return fmt.Errorf("storageProviderEndpoint does not contain a host entry")
			}

			if ip := net.ParseIP(storageProviderEndpointHost); ip == nil {
				loadBalancerEgressRule := ovnv1.EgressFirewallRule{}
				loadBalancerEgressRule.To.DNSName = storageProviderEndpointHost
				loadBalancerEgressRule.Type = ovnv1.EgressFirewallRuleAllow

				desired.Spec.Egress = append(
					[]ovnv1.EgressFirewallRule{
						loadBalancerEgressRule,
					},
					desired.Spec.Egress...,
				)
			}
		}

		if r.deadMansSnitchSecret.UID == "" {
			if err := r.get(r.deadMansSnitchSecret); err != nil {
				return fmt.Errorf("Unable to get DMS secret: %v", err)
			}
		}
		snitchStr := string(r.deadMansSnitchSecret.Data["SNITCH_URL"])
		if snitchStr == "" {
			return fmt.Errorf("DMS secret does not contain a SNITCH_URL entry")
		}
		snitchURL, err := url.Parse(snitchStr)
		if err != nil {
			return fmt.Errorf("Unable to parse DMS url: %v", err)
		}
		if snitchURL.Hostname() == "" {
			return fmt.Errorf("DMS snitch url %q does not have a valid hostname", snitchURL)
		}

		if r.smtpSecret.UID == "" {
			if err := r.get(r.smtpSecret); err != nil {
				return fmt.Errorf("Unable to get SMTP secret: %v", err)
			}
		}
		smtpHost := string(r.smtpSecret.Data["host"])
		if smtpHost == "" {
			return fmt.Errorf("smtp secret does not contain a host entry")
		}

		rhobsURL, err := url.Parse(r.RHOBSEndpoint)
		if err != nil {
			return fmt.Errorf("Unable to parse RHOBS url: %v", err)
		}
		if rhobsURL.Hostname() == "" {
			return fmt.Errorf("RHOBS url %q does not have a valid hostname", rhobsURL)
		}

		ssoURL, err := url.Parse(r.RHSSOTokenEndpoint)
		if err != nil {
			return fmt.Errorf("Unable to parse RHSSOTokenEndpoint url: %v", err)
		}
		if ssoURL.Hostname() == "" {
			return fmt.Errorf("RHSSOTokenEndpoint url %q does not have a valid hostname", ssoURL)
		}

		dmsEgressRule := ovnv1.EgressFirewallRule{}
		dmsEgressRule.To.DNSName = snitchURL.Hostname()
		dmsEgressRule.Type = ovnv1.EgressFirewallRuleAllow

		smtpEgressRule := ovnv1.EgressFirewallRule{}
		smtpEgressRule.To.DNSName = smtpHost
		smtpEgressRule.Type = ovnv1.EgressFirewallRuleAllow

		rhobsEgressRule := ovnv1.EgressFirewallRule{}
		rhobsEgressRule.To.DNSName = rhobsURL.Hostname()
		rhobsEgressRule.Type = ovnv1.EgressFirewallRuleAllow

		ssoEgressRule := ovnv1.EgressFirewallRule{}
		ssoEgressRule.To.DNSName = ssoURL.Hostname()
		ssoEgressRule.Type = ovnv1.EgressFirewallRuleAllow

		desired.Spec.Egress = append(
			[]ovnv1.EgressFirewallRule{
				dmsEgressRule,
				smtpEgressRule,
				rhobsEgressRule,
				ssoEgressRule,
			},
			desired.Spec.Egress...,
		)
		r.egressFirewall.Spec = desired.Spec
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to update egressFirewall: %v", err)
	}
	return nil

}

func (r *ManagedOCSReconciler) reconcileIngressNetworkPolicy() error {
	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.ingressNetworkPolicy, func() error {
		if err := r.own(r.ingressNetworkPolicy); err != nil {
			return err
		}
		desired := templates.NetworkPolicyTemplate.DeepCopy()
		r.ingressNetworkPolicy.Spec = desired.Spec
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to update ingress NetworkPolicy: %v", err)
	}
	return nil
}

func (r *ManagedOCSReconciler) reconcileCephIngressNetworkPolicy() error {
	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.cephIngressNetworkPolicy, func() error {
		if err := r.own(r.cephIngressNetworkPolicy); err != nil {
			return err
		}
		desired := templates.CephNetworkPolicyTemplate.DeepCopy()
		r.cephIngressNetworkPolicy.Spec = desired.Spec
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to update ceph ingress NetworkPolicy: %v", err)
	}
	return nil
}

func (r *ManagedOCSReconciler) reconcileProviderAPIServerNetworkPolicy() error {
	if r.DeploymentType != providerDeploymentType {
		r.Log.Info("Non provider deployment, skipping reconcile for api server ingress policy")
		return nil
	}
	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.providerAPIServerNetworkPolicy, func() error {
		if err := r.own(r.providerAPIServerNetworkPolicy); err != nil {
			return err
		}
		desired := templates.ProviderApiServerNetworkPolicyTemplate.DeepCopy()
		r.providerAPIServerNetworkPolicy.Spec = desired.Spec
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to update provider api server NetworkPolicy: %v", err)
	}
	return nil
}

func (r *ManagedOCSReconciler) reconcilePrometheusProxyNetworkPolicy() error {
	r.Log.Info("reconciling PrometheusProxyNetworkPolicy resources")

	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, r.prometheusProxyNetworkPolicy, func() error {
		if err := r.own(r.prometheusProxyNetworkPolicy); err != nil {
			return err
		}
		desired := templates.PrometheusProxyNetworkPolicyTemplate.DeepCopy()
		r.prometheusProxyNetworkPolicy.Spec = desired.Spec
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to update prometheus proxy NetworkPolicy: %v", err)
	}
	return nil
}

func (r *ManagedOCSReconciler) checkUninstallCondition() bool {
	configmap := &corev1.ConfigMap{}
	configmap.Name = r.AddonConfigMapName
	configmap.Namespace = r.namespace

	err := r.get(configmap)
	if err != nil {
		if !errors.IsNotFound(err) {
			r.Log.Error(err, "Unable to get addon delete configmap")
		}
		return false
	}
	_, ok := configmap.Labels[r.AddonConfigMapDeleteLabelKey]
	return ok
}

func (r *ManagedOCSReconciler) areComponentsReadyForUninstall() bool {
	subComponents := r.managedOCS.Status.Components
	return subComponents.StorageCluster.State == v1.ComponentReady &&
		subComponents.Prometheus.State == v1.ComponentReady &&
		subComponents.Alertmanager.State == v1.ComponentReady
}

func (r *ManagedOCSReconciler) hasOCSVolumes() (bool, error) {
	// get all the storage class
	storageClassList := storagev1.StorageClassList{}
	if err := r.UnrestrictedClient.List(r.ctx, &storageClassList); err != nil {
		return false, fmt.Errorf("unable to list storage classes: %v", err)
	}

	// create a set of storage class names who are using the OCS provisioner
	// provisioner name prefixed with the namespace name
	ocsStorageClass := make(map[string]bool)
	for i := range storageClassList.Items {
		storageClass := &storageClassList.Items[i]
		if strings.HasPrefix(storageClassList.Items[i].Provisioner, r.namespace) {
			ocsStorageClass[storageClass.Name] = true
		}
	}

	// get all the PVs
	pvList := &corev1.PersistentVolumeList{}
	if err := r.UnrestrictedClient.List(r.ctx, pvList); err != nil {
		return false, fmt.Errorf("unable to list persistent volumes: %v", err)
	}

	// check if there are any PVs using OCS storage classes
	for i := range pvList.Items {
		scName := pvList.Items[i].Spec.StorageClassName
		if ocsStorageClass[scName] {
			return true, nil
		}
	}
	return false, nil
}

func (r *ManagedOCSReconciler) hasOCSStorageConsumers() (bool, error) {
	storageConsumerList := ocsv1alpha1.StorageConsumerList{}
	if err := r.Client.List(r.ctx, &storageConsumerList, &client.ListOptions{Limit: int64(1)}); err != nil {
		return false, fmt.Errorf("unable to list storage consumers: %v", err)
	}
	return len(storageConsumerList.Items) > 0, nil
}

func (r *ManagedOCSReconciler) reconcileCSV() error {
	r.Log.Info("Reconciling CSVs")

	csvList := opv1a1.ClusterServiceVersionList{}
	if err := r.list(&csvList); err != nil {
		return fmt.Errorf("unable to list csv resources: %v", err)
	}

	for index := range csvList.Items {
		csv := &csvList.Items[index]
		if strings.HasPrefix(csv.Name, ocsOperatorName) {
			if err := r.updateOCSCSV(csv); err != nil {
				return fmt.Errorf("Failed to update OCS CSV: %v", err)
			}
		} else if strings.HasPrefix(csv.Name, mcgOperatorName) {
			if err := r.updateMCGCSV(csv); err != nil {
				return fmt.Errorf("Failed to update MCG CSV: %v", err)
			}
		}
	}
	return nil
}

func (r *ManagedOCSReconciler) updateOCSCSV(csv *opv1a1.ClusterServiceVersion) error {
	isChanged := false
	deployments := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
	for i := range deployments {
		containers := deployments[i].Spec.Template.Spec.Containers
		for j := range containers {
			switch container := &containers[j]; container.Name {
			case "ocs-operator":
				resources := utils.GetResourceRequirements("ocs-operator")
				if !equality.Semantic.DeepEqual(container.Resources, resources) {
					container.Resources = resources
					isChanged = true
				}
			case "rook-ceph-operator":
				resources := utils.GetResourceRequirements("rook-ceph-operator")
				if !equality.Semantic.DeepEqual(container.Resources, resources) {
					container.Resources = resources
					isChanged = true
				}
			case "ocs-metrics-exporter":
				resources := utils.GetResourceRequirements("ocs-metrics-exporter")
				if !equality.Semantic.DeepEqual(container.Resources, resources) {
					container.Resources = resources
					isChanged = true
				}
			default:
				r.Log.V(-1).Info("Could not find resource requirement", "Resource", container.Name)
			}
		}
	}
	if isChanged {
		if err := r.update(csv); err != nil {
			return fmt.Errorf("Failed to update OCS CSV: %v", err)
		}
	}
	return nil
}

func (r *ManagedOCSReconciler) updateMCGCSV(csv *opv1a1.ClusterServiceVersion) error {
	isChanged := false
	mcgDeployments := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
	for i := range mcgDeployments {
		deployment := &mcgDeployments[i]
		// Disable noobaa operator by scaling down the replica of noobaa deploymnet
		// in MCG Operator CSV.
		if deployment.Name == "noobaa-operator" &&
			(deployment.Spec.Replicas == nil || *deployment.Spec.Replicas > 0) {
			zero := int32(0)
			deployment.Spec.Replicas = &zero
			isChanged = true
		}
	}
	if isChanged {
		if err := r.update(csv); err != nil {
			return fmt.Errorf("Failed to update MCG CSV: %v", err)
		}
	}
	return nil
}

func (r *ManagedOCSReconciler) removeOLMComponents() error {

	r.Log.Info("deleting deployer csv")
	if err := r.deleteCSVByPrefix(deployerCSVPrefix); err != nil {
		return fmt.Errorf("Unable to delete csv: %v", err)
	} else {
		r.Log.Info("Deployer csv removed successfully")
		return nil
	}
}

func (r *ManagedOCSReconciler) get(obj client.Object) error {
	key := client.ObjectKeyFromObject(obj)
	return r.Client.Get(r.ctx, key, obj)
}

func (r *ManagedOCSReconciler) list(obj client.ObjectList) error {
	listOptions := client.InNamespace(r.namespace)
	return r.Client.List(r.ctx, obj, listOptions)
}

func (r *ManagedOCSReconciler) update(obj client.Object) error {
	return r.Client.Update(r.ctx, obj)
}

func (r *ManagedOCSReconciler) delete(obj client.Object) error {
	if err := r.Client.Delete(r.ctx, obj); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *ManagedOCSReconciler) own(resource metav1.Object) error {
	// Ensure managedOCS ownership on a resource
	if err := ctrl.SetControllerReference(r.managedOCS, resource, r.Scheme); err != nil {
		return err
	}
	return nil
}

func (r *ManagedOCSReconciler) deleteCSVByPrefix(name string) error {
	if csv, err := r.getCSVByPrefix(name); err == nil {
		return r.delete(csv)
	} else if errors.IsNotFound(err) {
		return nil
	} else {
		return err
	}
}

func (r *ManagedOCSReconciler) getCSVByPrefix(name string) (*opv1a1.ClusterServiceVersion, error) {
	csvList := opv1a1.ClusterServiceVersionList{}
	if err := r.list(&csvList); err != nil {
		return nil, fmt.Errorf("unable to list csv resources: %v", err)
	}

	for index := range csvList.Items {
		candidate := &csvList.Items[index]
		if strings.HasPrefix(candidate.Name, name) {
			return candidate, nil
		}
	}
	return nil, errors.NewNotFound(opv1a1.Resource("csv"), fmt.Sprintf("unable to find a csv prefixed with %s", name))
}

func findStorageDeviceSet(storageDeviceSets []ocsv1.StorageDeviceSet, deviceSetName string) *ocsv1.StorageDeviceSet {
	for index := range storageDeviceSets {
		item := &storageDeviceSets[index]
		if item.Name == deviceSetName {
			return item
		}
	}
	return nil
}

func (r *ManagedOCSReconciler) ensureMCGDeployment(storageCluster *ocsv1.StorageCluster, mcgEnable bool) error {
	// Check and enable MCG in Storage Cluster spec
	if mcgEnable {
		r.Log.Info("Enabling Multi Cloud Gateway")
		storageCluster.Spec.MultiCloudGateway.ReconcileStrategy = "manage"
	} else if storageCluster.Spec.MultiCloudGateway.ReconcileStrategy == "manage" {
		r.Log.V(-1).Info("Trying to disable Multi Cloud Gateway, Invalid operation")
	}
	return nil
}

func (r *ManagedOCSReconciler) setDeviceSetCount(deviceSet *ocsv1.StorageDeviceSet, desiredDeviceSetCount int, currDeviceSetCount int) {
	r.Log.Info("Setting storage device set count", "Current", currDeviceSetCount, "New", desiredDeviceSetCount)
	if currDeviceSetCount <= desiredDeviceSetCount {
		deviceSet.Count = desiredDeviceSetCount
	} else {
		r.Log.V(-1).Info("Requested storage device set count will result in downscaling, which is not supported. Skipping")
		deviceSet.Count = currDeviceSetCount
	}
}

func (r *ManagedOCSReconciler) reconcileODFSubscription() error {
	subList := &opv1a1.SubscriptionList{}
	if err := r.list(subList); err != nil {
		return fmt.Errorf("Unable to list subscriptions %v", err)
	}
	var odfSub *opv1a1.Subscription = nil
	for index := range subList.Items {
		item := &subList.Items[index]
		if item.Spec.Package == odfOLMPackageName {
			odfSub = item
			break
		}
	}
	if odfSub == nil {
		return fmt.Errorf("Failed to find package name %s", odfOLMPackageName)
	}
	if odfSub.Spec.Channel == desiredODFSubscriptionChannel {
		return nil
	}
	r.Log.Info("Initiating ODF minor version upgrade by switiching channels", "current channel", odfSub.Spec.Channel, "new channel", desiredODFSubscriptionChannel)
	odfSub.Spec.Channel = desiredODFSubscriptionChannel

	if err := r.update(odfSub); err != nil {
		return fmt.Errorf("Failed to update %s subscription %v", odfOLMPackageName, err)
	}
	return nil
}
