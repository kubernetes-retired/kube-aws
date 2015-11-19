package cluster

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type ClusterInfo struct {
	Name         string
	ControllerIP string
}

func (c *ClusterInfo) String() string {
	buf := new(bytes.Buffer)
	w := new(tabwriter.Writer)
	w.Init(buf, 0, 8, 0, '\t', 0)

	fmt.Fprintf(w, "Cluster Name:\t%s\n", c.Name)
	fmt.Fprintf(w, "Controller IP:\t%s\n", c.ControllerIP)

	w.Flush()
	return buf.String()
}

type TLSConfig struct {
	CACertFile string
	CACert     []byte

	APIServerCertFile string
	APIServerCert     []byte
	APIServerKeyFile  string
	APIServerKey      []byte

	WorkerCertFile string
	WorkerCert     []byte
	WorkerKeyFile  string
	WorkerKey      []byte

	AdminCertFile string
	AdminCert     []byte
	AdminKeyFile  string
	AdminKey      []byte
}

func New(cfg *Config, awsConfig *aws.Config) *Cluster {
	return &Cluster{
		cfg: cfg,
		aws: awsConfig,
	}
}

type Cluster struct {
	cfg *Config
	aws *aws.Config
}

func (c *Cluster) stackName() string {
	return c.cfg.ClusterName
}

func (c *Cluster) Create(tlsConfig *TLSConfig) error {
	parameters := []*cloudformation.Parameter{
		{
			ParameterKey:     aws.String(parClusterName),
			ParameterValue:   aws.String(c.stackName()),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String(parNameKeyName),
			ParameterValue:   aws.String(c.cfg.KeyName),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String(parArtifactURL),
			ParameterValue:   aws.String(c.cfg.ArtifactURL),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String(parCACert),
			ParameterValue:   aws.String(base64.StdEncoding.EncodeToString(tlsConfig.CACert)),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String(parAPIServerCert),
			ParameterValue:   aws.String(base64.StdEncoding.EncodeToString(tlsConfig.APIServerCert)),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String(parAPIServerKey),
			ParameterValue:   aws.String(base64.StdEncoding.EncodeToString(tlsConfig.APIServerKey)),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String(parWorkerCert),
			ParameterValue:   aws.String(base64.StdEncoding.EncodeToString(tlsConfig.WorkerCert)),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String(parWorkerKey),
			ParameterValue:   aws.String(base64.StdEncoding.EncodeToString(tlsConfig.WorkerKey)),
			UsePreviousValue: aws.Bool(true),
		},
	}

	if c.cfg.ReleaseChannel != "" {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:     aws.String(parNameReleaseChannel),
			ParameterValue:   aws.String(c.cfg.ReleaseChannel),
			UsePreviousValue: aws.Bool(true),
		})
	}

	if c.cfg.ControllerInstanceType != "" {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:     aws.String(parNameControllerInstanceType),
			ParameterValue:   aws.String(c.cfg.ControllerInstanceType),
			UsePreviousValue: aws.Bool(true),
		})
	}

	if c.cfg.ControllerRootVolumeSize > 0 {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:     aws.String(parNameControllerRootVolumeSize),
			ParameterValue:   aws.String(fmt.Sprintf("%d", c.cfg.ControllerRootVolumeSize)),
			UsePreviousValue: aws.Bool(true),
		})
	}

	if c.cfg.WorkerCount > 0 {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:     aws.String(parWorkerCount),
			ParameterValue:   aws.String(fmt.Sprintf("%d", c.cfg.WorkerCount)),
			UsePreviousValue: aws.Bool(true),
		})
	}

	if c.cfg.WorkerInstanceType != "" {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:     aws.String(parNameWorkerInstanceType),
			ParameterValue:   aws.String(c.cfg.WorkerInstanceType),
			UsePreviousValue: aws.Bool(true),
		})
	}

	if c.cfg.WorkerRootVolumeSize > 0 {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:     aws.String(parNameWorkerRootVolumeSize),
			ParameterValue:   aws.String(fmt.Sprintf("%d", c.cfg.WorkerRootVolumeSize)),
			UsePreviousValue: aws.Bool(true),
		})
	}

	if c.cfg.AvailabilityZone != "" {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:     aws.String(parAvailabilityZone),
			ParameterValue:   aws.String(c.cfg.AvailabilityZone),
			UsePreviousValue: aws.Bool(true),
		})
	}

	tmplURL := fmt.Sprintf("%s/template.json", c.cfg.ArtifactURL)
	return createStackAndWait(cloudformation.New(c.aws), c.stackName(), tmplURL, parameters)
}

func (c *Cluster) Info() (*ClusterInfo, error) {
	resources, err := getStackResources(cloudformation.New(c.aws), c.stackName())
	if err != nil {
		return nil, err
	}

	info, err := mapStackResourcesToClusterInfo(ec2.New(c.aws), resources)
	if err != nil {
		return nil, err
	}

	info.Name = c.cfg.ClusterName
	return info, nil
}

func (c *Cluster) Destroy() error {
	return destroyStack(cloudformation.New(c.aws), c.stackName())
}
