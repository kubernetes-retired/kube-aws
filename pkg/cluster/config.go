package cluster

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"

	"gopkg.in/yaml.v2"
)

var (
	DefaultClusterName = "kubernetes"
)

type Config struct {
	ClusterName              string `yaml:"clusterName"`
	ExternalDNSName          string `yaml:"externalDNSName"`
	KeyName                  string `yaml:"keyName"`
	Region                   string `yaml:"region"`
	AvailabilityZone         string `yaml:"availabilityZone"`
	ArtifactURL              string `yaml:"artifactURL"`
	ReleaseChannel           string `yaml:"releaseChannel"`
	ControllerInstanceType   string `yaml:"controllerInstanceType"`
	ControllerRootVolumeSize int    `yaml:"controllerRootVolumeSize"`
	WorkerCount              int    `yaml:"workerCount"`
	WorkerInstanceType       string `yaml:"workerInstanceType"`
	WorkerRootVolumeSize     int    `yaml:"workerRootVolumeSize"`
}

func (cfg *Config) Valid() error {
	if cfg.ExternalDNSName == "" {
		return errors.New("externalDNSName must be set")
	}
	if cfg.KeyName == "" {
		return errors.New("keyName must be set")
	}
	if cfg.Region == "" {
		return errors.New("region must be set")
	}
	if cfg.ClusterName == "" {
		return errors.New("clusterName must be set")
	}
	if _, err := url.Parse(cfg.ArtifactURL); err != nil {
		return fmt.Errorf("invalid artifactURL: %v", err)
	}
	return nil
}

func DecodeConfigFromFile(out *Config, loc string) error {
	d, err := ioutil.ReadFile(loc)
	if err != nil {
		return fmt.Errorf("failed reading config file: %v", err)
	}

	if err = yaml.Unmarshal(d, &out); err != nil {
		return fmt.Errorf("failed decoding config file: %v", err)
	}

	if err = out.Valid(); err != nil {
		return fmt.Errorf("config file invalid: %v", err)
	}

	return nil
}

func NewDefaultConfig(ver string) *Config {
	return &Config{
		ClusterName: "kubernetes",
		ArtifactURL: DefaultArtifactURL(ver),
	}
}

func DefaultArtifactURL(ver string) string {
	return fmt.Sprintf("https://coreos-kubernetes.s3.amazonaws.com/%s", ver)
}
