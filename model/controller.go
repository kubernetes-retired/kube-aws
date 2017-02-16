package model

import "errors"

// TODO Merge this with NodePoolConfig
type Controller struct {
	AutoScalingGroup   AutoScalingGroup  `yaml:"autoScalingGroup,omitempty"`
	ClusterAutoscaler  ClusterAutoscaler `yaml:"clusterAutoscaler,omitempty"`
	LoadBalancer       ControllerElb     `yaml:"loadBalancer,omitempty"`
	ManagedIamRoleName string            `yaml:"managedIamRoleName,omitempty"`
	Subnets            []Subnet          `yaml:"subnets,omitempty"`
}

func NewDefaultController() Controller {
	n := 1
	return Controller{
		AutoScalingGroup: AutoScalingGroup{RollingUpdateMinInstancesInService: &n},
	}
}

func (c Controller) LogicalName() string {
	return "Controllers"
}

func (c Controller) Validate() error {
	if err := c.AutoScalingGroup.Valid(); err != nil {
		return err
	}

	if c.ClusterAutoscaler.Enabled() {
		return errors.New("cluster-autoscaler can't be enabled for a control plane because " +
			"allowing so for a group of controller nodes spreading over 2 or more availability zones " +
			"results in unreliability while scaling nodes out.")
	}
	return nil
}

type ControllerElb struct {
	Private bool
	Subnets []Subnet
}
