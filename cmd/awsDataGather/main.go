package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/red-hat-storage/ocs-osd-deployer/pkg/aws"
	"github.com/red-hat-storage/ocs-osd-deployer/utils"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	log := ctrl.Log.WithName("main")

	mainContext := context.Background()

	namespace, found := os.LookupEnv("NAMESPACE")
	if !found {
		fmt.Fprintf(os.Stderr,
			"NAMESPACE environment variable not found\n")
		os.Exit(1)
	}
	podName, found := os.LookupEnv("POD_NAME")
	if !found {
		fmt.Fprintf(os.Stderr,
			"POD_NAME environment variable not found\n")
		os.Exit(1)
	}

	// We need the deployment name to use as an owner reference.
	// However, the deployment name cannot be passed as an environment variable,
	// so we derive it from the pod name.
	deploymentName, err := utils.DeploymentNameFromPodName(podName)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Could not determine deployment name from pod name (%s)",
			podName)
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

	// This will later be used as the OwnerReference for the data ConfigMap
	var deployment appsv1.Deployment
	err = k8sClient.Get(mainContext, client.ObjectKey{
		Name:      deploymentName,
		Namespace: namespace,
	}, &deployment)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to find deployment '%s' in namespace '%s'", deploymentName, namespace))
		os.Exit(1)
	}

	log.Info("Gathering AWS data")
	if err := gatherAndSaveData(aws.IMDSv1Server, deployment, k8sClient, mainContext); err != nil {
		log.Error(err, "Failed to gather AWS data")
		os.Exit(1)
	}

	log.Info("AWS data gathering successfully completed!")

	// Set up signal handler, so that a clean up procedure will be run if the pod running this code is deleted.
	// This code is run as part of a deployment that is pushed to Kubernetes through an OLM Bundle.
	// At this time, OLM Bundles do not support restartPolicy: OnFailure,
	// so terminating this program will cause the pod to keep getting restarted and needlessly running.
	// To avoid this, a signal handler is set up to only terminate the program when the pod is getting deleted.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan

	log.Info("AWS data gather program terminating")
}

func gatherAndSaveData(imdsServer string, deployment appsv1.Deployment, k8sClient client.Client, context context.Context) error {
	log := ctrl.Log.WithName("GatherData")

	imdsClient := aws.NewIMDSClient(imdsServer, k8sClient, context)

	log.Info("Fetching AWS VPC IPv4 CIDR data")
	if err := imdsClient.FetchIPv4CIDR(); err != nil {
		return fmt.Errorf("Failed to get VPC IPv4 CIDR: %v", err)
	}

	log.Info("Creating Config Map with AWS data")
	// Setting the owner of the configmap resource to this deployment that creates it.
	// That way, the configmap will go away when the deployment does.
	// The version and kind are being manually inserted because the deployment struct doesn't have it.
	// See: https://github.com/kubernetes/client-go/issues/861#issuecomment-686806279
	owner := metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       deployment.Name,
		UID:        deployment.UID,
	}
	if err := imdsClient.WriteToConfigMap(deployment.Namespace, owner); err != nil {
		return fmt.Errorf("Failed to create configmap: %v", err)
	}

	return nil
}
