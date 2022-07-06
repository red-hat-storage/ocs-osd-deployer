package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/red-hat-storage/ocs-osd-deployer/pkg/aws"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	log := ctrl.Log.WithName("aws-data-gather")

	namespace, found := os.LookupEnv("NAMESPACE")
	if !found {
		fmt.Fprintf(os.Stderr,
			"NAMESPACE environment variable not found\n")
		os.Exit(1)
	}

	log.Info("Setting up k8s client")
	var options client.Options
	options.Scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(options.Scheme))

	k8sClient, err := client.New(config.GetConfigOrDie(), options)
	if err != nil {
		log.Error(err, "error creating client")
		os.Exit(1)
	}

	err = aws.GatherData(aws.IMDSv1Server, k8sClient, namespace, log)
	if err != nil {
		log.Error(err, "error running aws data gather")
		os.Exit(1)
	}

	log.Info("AWS data gathering successfully completed!")

	// Set up signal handler, so that a clean up procedure will be run if the pod running this code is deleted.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan

	log.Info("AWS data gather program terminating")
}
