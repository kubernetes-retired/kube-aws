package coreosutil

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func GetAMIData(channel string) (map[string]map[string]string, error) {
	r, err := http.Get(fmt.Sprintf("https://coreos.com/dist/aws/aws-%s.json", channel))
	if err != nil {
		return nil, fmt.Errorf("failed to get AMI data: %s: %v", channel, err)
	}

	if r.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get AMI data: %s: invalid status code: %d", channel, r.StatusCode)
	}

	output := map[string]map[string]string{}

	err = json.NewDecoder(r.Body).Decode(&output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AMI data: %s: %v", channel, err)
	}
	r.Body.Close()

	return output, nil
}
