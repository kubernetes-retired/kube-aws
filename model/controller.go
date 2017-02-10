package model

type Controller struct {
	LoadBalancer     ControllerElb `yaml:"loadBalancer,omitempty"`
	AutoScalingGroup `yaml:"autoScalingGroup,omitempty"`
	Subnets          []Subnet `yaml:"subnets,omitempty"`
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

type ControllerElb struct {
	Private bool
	Subnets []Subnet
}
