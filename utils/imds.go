package utils

import (
	"fmt"
	"time"
)

const (
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
	IMDSv1Server      = "http://169.254.169.254"
	DataConfigMapName = "aws-data"
	CIDRKey           = "vpc-cidr"
)

func IMDSFetchIPv4CIDR(imdsServerAddr string) (string, error) {
	// This method needs an instance mac address to get the VPC ipv4 CIDR.
	var mac string
	err := Retry(3, 1*time.Second, func() error {
		var err error
		endpoint := fmt.Sprintf("%s/latest/meta-data/mac", imdsServerAddr)
		if mac, err = HTTPGetAndParseBody(endpoint); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to determine mac address of instance")
	}

	var cidr string
	err = Retry(3, 1*time.Second, func() error {
		var err error
		endpoint := fmt.Sprintf("%s/latest/meta-data/network/interfaces/macs/%s/vpc-ipv4-cidr-block",
			imdsServerAddr, mac)
		if cidr, err = HTTPGetAndParseBody(endpoint); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("Could not get VPC CIDR using mac address %q: %v", mac, err)
	}

	return cidr, nil
}
