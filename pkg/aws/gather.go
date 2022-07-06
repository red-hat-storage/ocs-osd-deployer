package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/red-hat-storage/ocs-osd-deployer/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
	IMDSv1Server      = "http://169.254.169.254"
	DataConfigMapName = "aws-data"
	CidrKey           = "vpc-cidr"
)

func GatherData(imdsServerAddr string, k8sClient client.Client, namespace string, log logr.Logger) error {
	// A mac address for one of the interfaces on the instance is needed
	// to query for the VPC CIDR block.
	log.Info("Determining mac address on instance")
	var mac string
	err := utils.Retry(3, time.Second, func() error {
		var err error
		endpoint := fmt.Sprintf("%s/latest/meta-data/mac", imdsServerAddr)
		if mac, err = utils.HTTPGetAndParseBody(endpoint); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find mac address of instance")
	}
	log.Info("MAC address found.", "mac", mac)

	log.Info("Determining VPC CIDR")
	var cidr string
	err = utils.Retry(3, time.Second, func() error {
		var err error
		endpoint := fmt.Sprintf("%s/latest/meta-data/network/interfaces/macs/%s/vpc-ipv4-cidr-block",
			imdsServerAddr, mac)
		if cidr, err = utils.HTTPGetAndParseBody(endpoint); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to find VPC CIDR using mac address %q", mac)
	}
	log.Info("VPC CIDR found.", "vpc-cidr", cidr)

	log.Info("Creating Config Map with AWS data")
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DataConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{},
	}
	_, err = ctrl.CreateOrUpdate(context.Background(), k8sClient, &configMap, func() error {
		configMap.Data[CidrKey] = cidr
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to create configmap: %v", err)
	}
	return nil
}
