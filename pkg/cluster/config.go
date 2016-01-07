package cluster

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"

	"gopkg.in/yaml.v2"
)

const (
	DefaultVPCCIDR             = "10.0.0.0/16"
	DefaultInstanceCIDR        = "10.0.0.0/24"
	DefaultControllerIP        = "10.0.0.50"
	DefaultPodCIDR             = "10.2.0.0/16"
	DefaultServiceCIDR         = "10.3.0.0/24"
	DefaultKubernetesServiceIP = "10.3.0.1"
	DefaultDNSServiceIP        = "10.3.0.10"
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
	VPCCIDR                  string `yaml:"vpcCIDR"`
	InstanceCIDR             string `yaml:"instanceCIDR"`
	ControllerIP             string `yaml:"controllerIP"`
	PodCIDR                  string `yaml:"podCIDR"`
	ServiceCIDR              string `yaml:"serviceCIDR"`
	KubernetesServiceIP      string `yaml:"kubernetesServiceIP"`
	DNSServiceIP             string `yaml:"dnsServiceIP"`
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

	_, vpcNet, err := net.ParseCIDR(cfg.VPCCIDR)
	if err != nil {
		return fmt.Errorf("invalid vpcCIDR: %v", err)
	}

	instancesNetIP, instancesNet, err := net.ParseCIDR(cfg.InstanceCIDR)
	if err != nil {
		return fmt.Errorf("invalid instanceCIDR: %v", err)
	}
	if !vpcNet.Contains(instancesNetIP) {
		return fmt.Errorf("vpcCIDR (%s) does not contain instanceCIDR (%s)",
			cfg.VPCCIDR,
			cfg.InstanceCIDR,
		)
	}

	controllerIPAddr := net.ParseIP(cfg.ControllerIP)
	if controllerIPAddr == nil {
		return fmt.Errorf("invalid controllerIP: %s", cfg.ControllerIP)
	}
	if !instancesNet.Contains(controllerIPAddr) {
		return fmt.Errorf("instanceCIDR (%s) does not contain controllerIP (%s)",
			cfg.InstanceCIDR,
			cfg.ControllerIP,
		)
	}

	podNetIP, podNet, err := net.ParseCIDR(cfg.PodCIDR)
	if err != nil {
		return fmt.Errorf("invalid podCIDR: %v", err)
	}
	if vpcNet.Contains(podNetIP) {
		return fmt.Errorf("vpcCIDR (%s) overlaps with podCIDR (%s)", cfg.VPCCIDR, cfg.PodCIDR)
	}

	serviceNetIP, serviceNet, err := net.ParseCIDR(cfg.ServiceCIDR)
	if err != nil {
		return fmt.Errorf("invalid serviceCIDR: %v", err)
	}
	if vpcNet.Contains(serviceNetIP) {
		return fmt.Errorf("vpcCIDR (%s) overlaps with serviceCIDR (%s)", cfg.VPCCIDR, cfg.ServiceCIDR)
	}
	if podNet.Contains(serviceNetIP) || serviceNet.Contains(podNetIP) {
		return fmt.Errorf("serviceCIDR (%s) overlaps with podCIDR (%s)", cfg.ServiceCIDR, cfg.PodCIDR)
	}

	kubernetesServiceIPAddr := net.ParseIP(cfg.KubernetesServiceIP)
	if kubernetesServiceIPAddr == nil {
		return fmt.Errorf("Invalid kubernetesServiceIP: %s", cfg.KubernetesServiceIP)
	}
	if !serviceNet.Contains(kubernetesServiceIPAddr) {
		return fmt.Errorf("serviceCIDR (%s) does not contain kubernetesServiceIP (%s)", cfg.ServiceCIDR, cfg.KubernetesServiceIP)
	}

	dnsServiceIPAddr := net.ParseIP(cfg.DNSServiceIP)
	if dnsServiceIPAddr == nil {
		return fmt.Errorf("Invalid dnsServiceIP: %s", cfg.DNSServiceIP)
	}
	if !serviceNet.Contains(dnsServiceIPAddr) {
		return fmt.Errorf("serviceCIDR (%s) does not contain dnsServiceIP (%s)", cfg.ServiceCIDR, cfg.DNSServiceIP)
	}

	return nil
}

func DecodeConfigFromFile(out *Config, loc string) error {
	d, err := ioutil.ReadFile(loc)
	if err != nil {
		return fmt.Errorf("failed reading config file: %v", err)
	}

	return decodeConfigBytes(out, d)
}

func decodeConfigBytes(out *Config, d []byte) error {
	if err := yaml.Unmarshal(d, &out); err != nil {
		return fmt.Errorf("failed decoding config file: %v", err)
	}

	if err := out.Valid(); err != nil {
		return fmt.Errorf("config file invalid: %v", err)
	}

	return nil
}

func NewDefaultConfig(ver string) *Config {
	return &Config{
		ClusterName:         "kubernetes",
		ArtifactURL:         DefaultArtifactURL(ver),
		VPCCIDR:             DefaultVPCCIDR,
		InstanceCIDR:        DefaultInstanceCIDR,
		ControllerIP:        DefaultControllerIP,
		PodCIDR:             DefaultPodCIDR,
		ServiceCIDR:         DefaultServiceCIDR,
		KubernetesServiceIP: DefaultKubernetesServiceIP,
		DNSServiceIP:        DefaultDNSServiceIP,
	}
}

func DefaultArtifactURL(ver string) string {
	return fmt.Sprintf("https://coreos-kubernetes.s3.amazonaws.com/%s", ver)
}
