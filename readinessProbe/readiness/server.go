package readiness

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	v1 "github.com/red-hat-storage/ocs-osd-deployer/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	listenAddr          string = ":8081"
	readinessPath       string = "/readyz/"
	NamespaceEnvVarName string = "NAMESPACE"
)

func isReady(client client.Client, managedOCSResource types.NamespacedName) (bool, error) {

	var managedOCS v1.ManagedOCS

	if err := client.Get(context.Background(), managedOCSResource, &managedOCS); err != nil {
		return false, err
	}

	ready := managedOCS.Status.Components.StorageCluster.State == v1.ComponentReady &&
		managedOCS.Status.Components.Prometheus.State == v1.ComponentReady &&
		managedOCS.Status.Components.Alertmanager.State == v1.ComponentReady &&
		managedOCS.Status.Components.ODF.State == v1.ComponentReady

	return ready, nil
}

func RunServer(client client.Client, managedOCSResource types.NamespacedName, log logr.Logger) error {

	// Readiness probe is defined here.
	// From k8s documentation:
	// "Any code greater than or equal to 200 and less than 400 indicates success."
	// [indicates that the deployment is ready]
	// "Any other code indicates failure."
	// [indicates that the deployment is not ready]
	http.HandleFunc(readinessPath, func(httpw http.ResponseWriter, req *http.Request) {
		ready, err := isReady(client, managedOCSResource)

		if err != nil {
			log.Error(err, "error checking readiness\n")
			httpw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if ready {
			httpw.WriteHeader(http.StatusOK)
		} else {
			httpw.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	return http.ListenAndServe(listenAddr, nil)
}
