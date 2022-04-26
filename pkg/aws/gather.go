package aws

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IMDSv1Server      = "http://169.254.169.254"
	DataConfigMapName = "aws-data"
	CidrKey           = "vpc-cidr"
)

func GatherData(imdsServerAddr string, k8sClient client.Client, namespace string, log logr.Logger) error {
	// A mac address for one of the interfaces on the instance is needed
	// to query for the VPC CIDR block.
	log.Info("Determining mac address on instance")
	mac, err := httpGet(fmt.Sprintf("%s/%s", imdsServerAddr, "latest/meta-data/mac"), log)
	if err != nil {
		return fmt.Errorf("failed to find mac address of instance")
	}
	log.Info("MAC address found.", "mac", mac)

	log.Info("Determining VPC CIDR")
	cidr, err := httpGet(
		fmt.Sprintf("%s/%s/%s/%s",
			imdsServerAddr, "latest/meta-data/network/interfaces/macs", mac, "vpc-ipv4-cidr-block"), log)
	if err != nil {
		return fmt.Errorf("Failed to find VPC CIDR")
	}
	log.Info("VPC CIDR found.", "vpc-cidr", cidr)

	log.Info("Creating Config Map with AWS data")
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DataConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			CidrKey: cidr,
		},
	}
	err = createOrUpdateConfigMap(k8sClient, &configMap)
	if err != nil {
		return fmt.Errorf("Failed to create configmap: %v", err)
	}
	return nil
}

func createOrUpdateConfigMap(k8sClient client.Client, configMap *corev1.ConfigMap) error {
	err := k8sClient.Create(context.Background(), configMap)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			err = k8sClient.Update(context.Background(), configMap)
		}
	}
	return err
}

func httpGet(address string, log logr.Logger) (string, error) {
	// To be robust to network hiccups, this get function will retry on failures
	for tries := 0; tries < 3; tries++ {
		if tries > 0 {
			log.Info("\tRetrying...")
			// Sleep for a second
			time.Sleep(time.Second)
		}

		resp, err := http.Get(address)
		if err != nil {
			log.Error(err, "Failed to GET")
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error(err, "Failed to read body of GET")
			continue
		}
		return string(body), nil
	}
	return "", fmt.Errorf("http get call failed")
}
