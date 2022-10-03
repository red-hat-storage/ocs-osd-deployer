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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	openshiftv1 "github.com/openshift/api/network/v1"
	operators "github.com/operator-framework/api/pkg/operators/v1alpha1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1a1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	odfv1a1 "github.com/red-hat-data-services/odf-operator/api/v1alpha1"
	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v1"
	ocsv1alpha1 "github.com/red-hat-storage/ocs-operator/api/v1alpha1"
	v1 "github.com/red-hat-storage/ocs-osd-deployer/api/v1alpha1"
	"github.com/red-hat-storage/ocs-osd-deployer/controllers"
	// +kubebuilder:scaffold:imports
)

const (
	namespaceEnvVarName          = "NAMESPACE"
	addonNameEnvVarName          = "ADDON_NAME"
	sopEndpointEnvVarName        = "SOP_ENDPOINT"
	alertSMTPFromAddrEnvVarName  = "ALERT_SMTP_FROM_ADDR"
	deploymentTypeEnvVarName     = "DEPLOYMENT_TYPE"
	rhobsEndpointEnvVarName      = "RHOBS_ENDPOINT"
	rhssoTokenEndpointEnvVarName = "RH_SSO_TOKEN_ENDPOINT"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	addAllSchemes(scheme)

}

func addAllSchemes(scheme *runtime.Scheme) {

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(ocsv1.AddToScheme(scheme))

	utilruntime.Must(promv1.AddToScheme(scheme))

	utilruntime.Must(promv1a1.AddToScheme(scheme))

	utilruntime.Must(v1.AddToScheme(scheme))

	utilruntime.Must(operators.AddToScheme(scheme))

	utilruntime.Must(openshiftv1.AddToScheme(scheme))

	utilruntime.Must(odfv1a1.AddToScheme(scheme))

	utilruntime.Must(ocsv1alpha1.AddToScheme(scheme))

	utilruntime.Must(configv1.AddToScheme(scheme))

	utilruntime.Must(consolev1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.StacktraceLevel(zapcore.ErrorLevel)))

	envVars, err := readEnvVars()
	if err != nil {
		setupLog.Error(err, "Failed to get environment variables")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "e0c63ac0.openshift.io",
		Namespace:          envVars[namespaceEnvVarName],
	})
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	addonName := envVars[addonNameEnvVarName]
	if err = (&controllers.ManagedOCSReconciler{
		Client:                       mgr.GetClient(),
		UnrestrictedClient:           getUnrestrictedClient(),
		Log:                          ctrl.Log.WithName("controllers").WithName("ManagedOCS"),
		Scheme:                       mgr.GetScheme(),
		AddonParamSecretName:         fmt.Sprintf("addon-%v-parameters", addonName),
		AddonConfigMapName:           addonName,
		AddonConfigMapDeleteLabelKey: fmt.Sprintf("api.openshift.com/addon-%v-delete", addonName),
		PagerdutySecretName:          fmt.Sprintf("%v-pagerduty", addonName),
		DeadMansSnitchSecretName:     fmt.Sprintf("%v-deadmanssnitch", addonName),
		SMTPSecretName:               fmt.Sprintf("%v-smtp", addonName),
		SOPEndpoint:                  envVars[sopEndpointEnvVarName],
		AlertSMTPFrom:                envVars[alertSMTPFromAddrEnvVarName],
		DeploymentType:               envVars[deploymentTypeEnvVarName],
		CustomerNotificationHTMLPath: "templates/customernotification.html",
		RHOBSSecretName:              fmt.Sprintf("%v-prom-remote-write", addonName),
		RHOBSEndpoint:                envVars[rhobsEndpointEnvVarName],
		RHSSOTokenEndpoint:           envVars[rhssoTokenEndpointEnvVarName],
	}).SetupWithManager(mgr, nil); err != nil {
		setupLog.Error(err, "Unable to create controller", "controller", "ManagedOCS")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := ensureManagedOCS(mgr.GetClient(), setupLog, envVars); err != nil {
		os.Exit(1)
	}

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}

// getUnrestrictedClient creates a client required for listing PVCs from all namespaces.
func getUnrestrictedClient() client.Client {
	var options client.Options

	options.Scheme = runtime.NewScheme()
	addAllSchemes(options.Scheme)
	k8sClient, err := client.New(config.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "error creating client")
		os.Exit(1)
	}
	return k8sClient
}

func readEnvVars() (map[string]string, error) {
	envVars := map[string]string{
		namespaceEnvVarName:          "",
		addonNameEnvVarName:          "",
		sopEndpointEnvVarName:        "",
		alertSMTPFromAddrEnvVarName:  "",
		deploymentTypeEnvVarName:     "",
		rhobsEndpointEnvVarName:      "",
		rhssoTokenEndpointEnvVarName: "",
	}
	for envVarName := range envVars {
		val, found := os.LookupEnv(envVarName)
		if !found {
			return nil, fmt.Errorf("%s environment variable must be set", envVarName)
		}
		envVars[envVarName] = val
	}

	return envVars, nil
}

func ensureManagedOCS(c client.Client, log logr.Logger, envVars map[string]string) error {
	err := c.Create(context.Background(), &v1.ManagedOCS{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "managedocs",
			Namespace:  envVars[namespaceEnvVarName],
			Finalizers: []string{controllers.ManagedOCSFinalizer},
		},
	})
	if err == nil {
		log.Info("ManagedOCS resource created")
		return nil

	} else if errors.IsAlreadyExists(err) {
		log.Info("ManagedOCS resource already exists")
		return nil

	} else {
		log.Error(err, "Unable to create ManagedOCS resource")
		return err
	}
}
