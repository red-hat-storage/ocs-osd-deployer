package utils

import (
	"fmt"
	"strings"
)

const (
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
	IMDSv1Server      = "http://169.254.169.254"
	IMDSConfigMapName = "aws-data"
	CIDRKey           = "vpc-cidr"
)

func IMDSFetchIPv4CIDR(imdsServerAddr string) (string, error) {
	// This method needs an instance mac address to get the VPC ipv4 CIDR.
	var mac string
	var err error
	endpoint := fmt.Sprintf("%s/latest/meta-data/mac", imdsServerAddr)
	mac, err = HTTPGetAndParseBody(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to determine mac address of instance: %v", err)
	}

	var cidrs string
	endpoint = fmt.Sprintf("%s/latest/meta-data/network/interfaces/macs/%s/vpc-ipv4-cidr-blocks",
		imdsServerAddr, mac)
	cidrs, err = HTTPGetAndParseBody(endpoint)
	if err != nil {
		return "", fmt.Errorf("Could not get VPC CIDR using mac address %q: %v", mac, err)
	}

	return strings.Replace(cidrs, "\n", ";", -1), nil
}
