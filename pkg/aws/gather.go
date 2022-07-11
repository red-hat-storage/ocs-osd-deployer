package aws

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/red-hat-storage/ocs-osd-deployer/utils"
)

const (
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
	IMDSv1Server      = "http://169.254.169.254"
	DataConfigMapName = "aws-data"
	CidrKey           = "vpc-cidr"
)

func GatherData(imdsServerAddr string, log logr.Logger) (map[string]string, error) {
	var data map[string]string = map[string]string{}

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
		return nil, fmt.Errorf("failed to find mac address of instance")
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
		return nil, fmt.Errorf("Failed to find VPC CIDR using mac address %q", mac)
	}
	log.Info("VPC CIDR found.", "vpc-cidr", cidr)

	data[CidrKey] = cidr

	return data, nil
}
