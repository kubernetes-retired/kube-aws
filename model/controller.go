package model

type Controller struct {
	AutoScalingGroup `yaml:"autoScalingGroup,omitempty"`
}

func (c Controller) LogicalName() string {
	return "Controllers"
}
