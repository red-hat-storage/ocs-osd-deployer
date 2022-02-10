// Description: This program creates a web server to verify if the managedOCS
//              resource is ready. It is used as a readiness probe by the
//              ocs-osd-deployer operator. For this to be set up, the following
//              sidecar should be added to the manager CSV:
//      - name: readinessServer
//        command:
//        - /readinessServer
//        image: ocs-osd-deployer:latest
//        readinessProbe:
//          httpGet:
//            path: /readyz
//            port: 8081
//          initialDelaySeconds: 5
//          periodSeconds: 10
package main

import (
	"fmt"
	"os"

	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v1"
	v1 "github.com/red-hat-storage/ocs-osd-deployer/api/v1alpha1"
	"github.com/red-hat-storage/ocs-osd-deployer/readinessProbe/readiness"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	// Setup logging
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	log := ctrl.Log.WithName("readiness")

	// Setup client options
	var options client.Options

	// The readiness must have these schemes to deserialize the k8s objects
	options.Scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(options.Scheme))
	utilruntime.Must(ocsv1.AddToScheme(options.Scheme))
	utilruntime.Must(v1.AddToScheme(options.Scheme))

	k8sClient, err := client.New(config.GetConfigOrDie(), options)
	if err != nil {
		log.Error(err, "error creating client")
		os.Exit(1)
	}

	namespace, found := os.LookupEnv(readiness.NamespaceEnvVarName)
	if !found {
		log.Error(fmt.Errorf("%q not set", readiness.NamespaceEnvVarName), "error in environment variables")
		os.Exit(2)
	}

	managedOCSResource := types.NamespacedName{
		Name:      "managedocs",
		Namespace: namespace,
	}

	log.Info("starting HTTP server...")
	err = readiness.RunServer(k8sClient, managedOCSResource, log)
	if err != nil {
		log.Error(err, "server error")
	}
	log.Info("HTTP server terminated.")
}
