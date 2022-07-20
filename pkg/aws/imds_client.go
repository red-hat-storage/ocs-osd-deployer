package aws

import (
	"context"
	"fmt"
	"time"

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

/*
The Instance MetaData Service client object provides methods
to query AWS for data pertaining to the instance the code is running on.

The code is expected to run in a Kubernetes cluster, and has the ability
to generate a ConfigMap with the data that has been fetched placed with defined keys.
*/
type IMDSClient struct {
	imdsServerAddr string
	k8sClient      client.Client
	data           map[string]string
}

func NewIMDSClient(imdsServerAddr string, k8sClient client.Client) IMDSClient {
	return IMDSClient{
		imdsServerAddr: imdsServerAddr,
		k8sClient:      k8sClient,
		data:           map[string]string{},
	}
}

func (client *IMDSClient) FetchIPv4CIDR() error {
	// This method needs an instance mac address to get the VPC ipv4 CIDR.
	var mac string
	err := utils.Retry(3, 1*time.Second, func() error {
		var err error
		endpoint := fmt.Sprintf("%s/latest/meta-data/mac", client.imdsServerAddr)
		if mac, err = utils.HTTPGetAndParseBody(endpoint); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to determine mac address of instance")
	}

	var cidr string
	err = utils.Retry(3, 1*time.Second, func() error {
		var err error
		endpoint := fmt.Sprintf("%s/latest/meta-data/network/interfaces/macs/%s/vpc-ipv4-cidr-block",
			client.imdsServerAddr, mac)
		if cidr, err = utils.HTTPGetAndParseBody(endpoint); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Could not get VPC CIDR using mac address %q: %v", mac, err)
	}

	client.data[CidrKey] = cidr

	return nil
}

func (client *IMDSClient) CreateConfigMap(namespace string, owners ...metav1.OwnerReference) error {
	configMap := corev1.ConfigMap{}
	configMap.Name = DataConfigMapName
	configMap.Namespace = namespace

	_, err := ctrl.CreateOrUpdate(context.Background(), client.k8sClient, &configMap, func() error {
		// Setting the owner of the configmap resource to this deployment that creates it.
		// That way, the configmap will go away when the deployment does.
		configMap.OwnerReferences = owners

		configMap.Data = client.data

		return nil
	})
	return err
}
