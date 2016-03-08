package config

import (
	"fmt"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/coreosutil"
)

var regions = []string{
	"ap-northeast-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"eu-central-1",
	"eu-west-1",
	"sa-east-1",
	"us-east-1",
	"us-gov-west-1",
	"us-west-1",
	"us-west-2",
}

var supportedChannels = []string{
	"alpha",
}

func getAMI(region, channel string) (string, error) {
	regionMap := map[string]map[string]string{}

	for _, channel := range supportedChannels {
		regions, err := coreosutil.GetAMIData(channel)

		if err != nil {
			return "", err
		}

		for region, amis := range regions {
			if region == "release_info" {
				continue
			}

			if _, ok := regionMap[region]; !ok {
				regionMap[region] = map[string]string{}
			}

			if ami, ok := amis["hvm"]; ok {
				regionMap[region][channel] = ami
			}
		}
	}

	if regionMap[region] == nil {
		return "", fmt.Errorf("could not get AMI for region %s", region)
	}

	ami := regionMap[region][channel]
	if ami == "" {
		return "", fmt.Errorf("could not get AMI in region %s for channel %s", region, channel)
	}

	return ami, nil
}
