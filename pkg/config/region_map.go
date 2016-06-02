package config

import (
	"fmt"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/coreosutil"
)

func getAMI(region, channel string) (string, error) {

	regions, err := coreosutil.GetAMIData(channel)

	if err != nil {
		return "", fmt.Errorf("error getting ami data for channel %s: %v", channel, err)
	}

	amis, ok := regions[region]
	if !ok {
		return "", fmt.Errorf("could not find region %s for channel %s", region, channel)
	}

	if ami, ok := amis["hvm"]; ok {
		return ami, nil
	}

	return "", fmt.Errorf("could not find hvm image for region %s, channel %s", region, channel)
}
