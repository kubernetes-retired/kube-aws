package model

import "strings"

type Subnet struct {
	AvailabilityZone  string     `yaml:"availabilityZone,omitempty"`
	InstanceCIDR      string     `yaml:"instanceCIDR,omitempty"`
	RouteTableID      string     `yaml:"routeTableId,omitempty"`
	NatGateway        NatGateway `yaml:"natGateway,omitempty"`
}

type NatGateway struct {
	ID              string `yaml:"id,omitempty"`
	EIPAllocationID string `yaml:"eipAllocationId,omitempty"`
}

func (c Subnet) LogicalName() string {
	return "Subnet" + strings.Replace(strings.Title(c.AvailabilityZone), "-", "", -1)
}
