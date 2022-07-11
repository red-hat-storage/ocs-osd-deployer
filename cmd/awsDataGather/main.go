package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/red-hat-storage/ocs-osd-deployer/pkg/aws"
	corev1 "k8s.io/api/core/v1"
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
	podName, found := os.LookupEnv("NAME")
	if !found {
		fmt.Fprintf(os.Stderr,
			"NAME environment variable not found\n")
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

	// Need to get the pod resource that is running
	// This will later be used as the ControllerReference for the data ConfigMap
	var pod corev1.Pod
	err = k8sClient.Get(context.Background(), client.ObjectKey{
		Name:      podName,
		Namespace: namespace,
	}, &pod)
	if err != nil {
		log.Error(err, "Failed to find pod '%s' in namespace '%s'", podName, namespace)
		os.Exit(1)
	}

	log.Info("Gathering AWS data")
	awsData, err := aws.GatherData(aws.IMDSv1Server, log)
	if err != nil {
		log.Error(err, "error running aws data gather")
		os.Exit(1)
	}

	log.Info("Creating Config Map with AWS data")
	configMap := corev1.ConfigMap{}
	configMap.Name = aws.DataConfigMapName
	configMap.Namespace = namespace
	_, err = ctrl.CreateOrUpdate(context.Background(), k8sClient, &configMap, func() error {
		if err := ctrl.SetControllerReference(&pod, &configMap, options.Scheme); err != nil {
			return err
		}
		configMap.Data = map[string]string{
			aws.CidrKey: awsData[aws.CidrKey],
		}
		return nil
	})
	if err != nil {
		log.Error(err, "Failed to create configmap")
		os.Exit(1)
	}

	log.Info("AWS data gathering successfully completed!")

	// Set up signal handler, so that a clean up procedure will be run if the pod running this code is deleted.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan

	log.Info("AWS data gather program terminating")
}
