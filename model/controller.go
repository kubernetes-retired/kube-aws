package model

type Controller struct {
	LoadBalancer     ControllerElb `yaml:"loadBalancer,omitempty"`
	AutoScalingGroup `yaml:"autoScalingGroup,omitempty"`
	Subnets          []Subnet `yaml:"subnets,omitempty"`
}

func (c Controller) LogicalName() string {
	return "Controllers"
}

type ControllerElb struct {
	Private bool
	Subnets []Subnet
}
