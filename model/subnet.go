package model

import (
	"strings"
)

type Subnet struct {
	//ID                string `yaml:"id,omitempty"`
	AvailabilityZone string     `yaml:"availabilityZone,omitempty"`
	InstanceCIDR     string     `yaml:"instanceCIDR,omitempty"`
	RouteTableID     string     `yaml:"routeTableId,omitempty"`
	NatGateway       NatGateway `yaml:"natGateway,omitempty"`
	TopLevel         bool
}

func (c Subnet) AvailabilityZoneLogicalName() string {
	return strings.Replace(strings.Title(c.AvailabilityZone), "-", "", -1)
}

func (c Subnet) LogicalName() string {
	if c.TopLevel == true {
		return "Subnet" + c.AvailabilityZoneLogicalName()
	}
	return "PrivateSubnet" + c.AvailabilityZoneLogicalName()
}

type NatGateway struct {
	ID              string `yaml:"id,omitempty"`
	EIPAllocationID string `yaml:"eipAllocationId,omitempty"`
}
