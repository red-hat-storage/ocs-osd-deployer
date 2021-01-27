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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/logr"
	ocsv1 "github.com/openshift/ocs-operator/pkg/apis"
	v1 "github.com/openshift/ocs-osd-deployer/api/v1alpha1"
	"github.com/openshift/ocs-osd-deployer/controllers"
	// +kubebuilder:scaffold:imports
)

const (
	namespaceEnvVar    = "NAMESPACE"
	configSecretEnvVar = "CONFIG"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(ocsv1.AddToScheme(scheme))

	utilruntime.Must(v1.AddToScheme(scheme))
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

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	namespace, err := getNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get namespace name")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "e0c63ac0.openshift.io",
		Namespace:          namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ManagedOCSReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("ManagedOCS"),
		Scheme:           mgr.GetScheme(),
		ConfigSecretName: getConfigName(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ManagedOCS")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := ensureManagedOCS(mgr.GetClient(), setupLog, namespace); err != nil {
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getNamespace() (string, error) {

	ns, found := os.LookupEnv(namespaceEnvVar)
	if !found {
		err := fmt.Errorf("%s must be set", namespaceEnvVar)
		return "", err
	}
	return ns, nil
}

// The name of the secret containing the addon params
func getConfigName() string {

	configName, found := os.LookupEnv(configSecretEnvVar)
	if !found {
		configName = controllers.AddonSecretName
	}
	return configName
}

func ensureManagedOCS(c client.Client, log logr.Logger, namespace string) error {
	err := c.Create(context.Background(), &v1.ManagedOCS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managedocs",
			Namespace: namespace,
		},
	})
	if err == nil {
		log.Info("ManagedOCS resource created")
		return nil

	} else if errors.IsAlreadyExists(err) {
		log.Info("ManagedOCS resource already exists")
		return nil

	} else {
		log.Error(err, "unable to create ManagedOCS resource")
		return err
	}
}
