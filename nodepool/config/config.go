package config

//go:generate go run ../../codegen/templates_gen.go DefaultClusterConfig=cluster.yaml StackTemplateTemplate=stack-template.json
//go:generate gofmt -w templates.go

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	cfg "github.com/coreos/kube-aws/config"
	"github.com/coreos/kube-aws/coreos/amiregistry"
	"github.com/coreos/kube-aws/coreos/userdatavalidation"
	"github.com/coreos/kube-aws/filereader/jsontemplate"
	"github.com/coreos/kube-aws/filereader/userdatatemplate"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Ref struct {
	PoolName string
}

type ComputedConfig struct {
	ProvidedConfig
	// Fields computed from Cluster
	AMI       string
	TLSConfig *cfg.CompactTLSAssets
}

type ProvidedConfig struct {
	cfg.KubeClusterSettings `yaml:",inline"`
	cfg.WorkerSettings      `yaml:",inline"`
	cfg.DeploymentSettings  `yaml:",inline"`
	EtcdEndpoints           string `yaml:"etcdEndpoints,omitempty"`
	NodePoolName            string `yaml:"nodePoolName,omitempty"`
	providedEncryptService  cfg.EncryptService
}

type StackTemplateOptions struct {
	WorkerTmplFile        string
	StackTemplateTmplFile string
	TLSAssetsDir          string
}

type stackConfig struct {
	*ComputedConfig
	UserDataWorker string
}

func (c ProvidedConfig) stackConfig(opts StackTemplateOptions, compressUserData bool) (*stackConfig, error) {
	assets, err := cfg.ReadTLSAssets(opts.TLSAssetsDir)
	if err != nil {
		return nil, err
	}
	stackConfig := stackConfig{}

	if stackConfig.ComputedConfig, err = c.Config(); err != nil {
		return nil, err
	}

	// TODO Cleaner way to inject this dependency
	var kmsSvc cfg.EncryptService
	if c.providedEncryptService != nil {
		kmsSvc = c.providedEncryptService
	} else {
		awsConfig := aws.NewConfig().
			WithRegion(stackConfig.ComputedConfig.Region).
			WithCredentialsChainVerboseErrors(true)

		kmsSvc = kms.New(session.New(awsConfig))
	}

	compactAssets, err := assets.Compact(stackConfig.ComputedConfig.KMSKeyARN, kmsSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to compress TLS assets: %v", err)
	}

	stackConfig.ComputedConfig.TLSConfig = compactAssets

	if stackConfig.UserDataWorker, err = userdatatemplate.GetString(opts.WorkerTmplFile, stackConfig.ComputedConfig, compressUserData); err != nil {
		return nil, fmt.Errorf("failed to render worker cloud config: %v", err)
	}

	return &stackConfig, nil
}

func (c ProvidedConfig) ValidateUserData(opts StackTemplateOptions) error {
	stackConfig, err := c.stackConfig(opts, false)
	if err != nil {
		return fmt.Errorf("failed to create stack config: %v", err)
	}

	err = userdatavalidation.Execute([]userdatavalidation.Entry{
		{"UserDataWorker", stackConfig.UserDataWorker},
	})

	return err
}

func (c ProvidedConfig) RenderStackTemplate(opts StackTemplateOptions) ([]byte, error) {
	stackConfig, err := c.stackConfig(opts, true)
	if err != nil {
		return nil, err
	}

	bytes, err := jsontemplate.GetBytes(opts.StackTemplateTmplFile, stackConfig)

	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func ClusterFromFile(filename string) (*ProvidedConfig, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c, err := ClusterFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("file %s: %v", filename, err)
	}

	return c, nil
}

func NewDefaultCluster() *ProvidedConfig {
	defaults := cfg.NewDefaultCluster()

	return &ProvidedConfig{
		DeploymentSettings: defaults.DeploymentSettings,
		WorkerSettings:     defaults.WorkerSettings,
	}
}

// ClusterFromBytes Necessary for unit tests, which store configs as hardcoded strings
func ClusterFromBytes(data []byte) (*ProvidedConfig, error) {
	c := NewDefaultCluster()
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("failed to parse cluster: %v", err)
	}

	// If the user specified no subnets, we assume that a single AZ configuration with the default instanceCIDR is demanded
	if len(c.Subnets) == 0 && c.InstanceCIDR == "" {
		c.InstanceCIDR = "10.0.1.0/24"
	}

	if err := c.valid(); err != nil {
		return nil, fmt.Errorf("invalid cluster: %v", err)
	}

	// For backward-compatibility
	if len(c.Subnets) == 0 {
		c.Subnets = []*cfg.Subnet{
			{
				AvailabilityZone: c.AvailabilityZone,
				InstanceCIDR:     c.InstanceCIDR,
			},
		}
	}

	return c, nil
}

func (c ProvidedConfig) Config() (*ComputedConfig, error) {
	config := ComputedConfig{ProvidedConfig: c}

	if c.AmiId == "" {
		var err error
		if config.AMI, err = amiregistry.GetAMI(config.Region, config.ReleaseChannel); err != nil {
			return nil, fmt.Errorf("failed getting AMI for config: %v", err)
		}
	} else {
		config.AMI = c.AmiId
	}

	return &config, nil
}

func (c ProvidedConfig) valid() error {
	if _, err := c.DeploymentSettings.Valid(); err != nil {
		return err
	}

	if _, err := c.KubeClusterSettings.Valid(); err != nil {
		return err
	}

	if err := c.WorkerSettings.Valid(); err != nil {
		return err
	}

	return nil
}

func (c ComputedConfig) VPCRef() string {
	//This means this VPC already exists, and we can reference it directly by ID
	if c.VPCID != "" {
		return fmt.Sprintf("%q", c.VPCID)
	}
	return fmt.Sprintf(`{"Fn::ImportValue" : {"Fn::Sub" : "%s-VPC"}}`, c.ClusterName)
}

func (c ComputedConfig) RouteTableRef() string {
	if c.RouteTableID != "" {
		return fmt.Sprintf("%q", c.RouteTableID)
	}
	return fmt.Sprintf(`{"Fn::ImportValue" : {"Fn::Sub" : "%s-RouteTable"}}`, c.ClusterName)
}

func (c ComputedConfig) WorkerSecurityGroupRefs() []string {
	return []string{
		// The security group assigned to worker nodes to allow communication to etcd nodes and controller nodes
		// which is created and maintained in the main cluster and then imported to node pools.
		fmt.Sprintf(`{"Fn::ImportValue" : {"Fn::Sub" : "%s-WorkerSecurityGroup"}}`, c.ClusterName),
	}
}
