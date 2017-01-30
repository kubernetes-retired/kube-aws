package model

import (
	"strings"
)

type Subnet struct {
	Identifier       `yaml:",inline"`
	CustomName       string           `yaml:"name,omitempty"`
	AvailabilityZone string           `yaml:"availabilityZone,omitempty"`
	InstanceCIDR     string           `yaml:"instanceCIDR,omitempty"`
	RouteTable       RouteTable       `yaml:"routeTable,omitempty"`
	NATGateway       NATGatewayConfig `yaml:"natGateway,omitempty"`
	InternetGateway  InternetGateway  `yaml:"internetGateway,omitempty"`
	Private          bool
}

func NewPublicSubnet(az string, cidr string) Subnet {
	return Subnet{
		AvailabilityZone: az,
		InstanceCIDR:     cidr,
		Private:          false,
	}
}

func NewPrivateSubnet(az string, cidr string) Subnet {
	return Subnet{
		AvailabilityZone: az,
		InstanceCIDR:     cidr,
		Private:          true,
	}
}

func NewExistingPrivateSubnet(az string, id string) Subnet {
	return Subnet{
		Identifier: Identifier{
			ID: id,
		},
		AvailabilityZone: az,
		Private:          true,
	}
}

func NewExistingPrivateSubnetWithPreconfiguredNATGateway(az string, id string, rtb string) Subnet {
	return Subnet{
		Identifier: Identifier{
			ID: id,
		},
		AvailabilityZone: az,
		Private:          true,
		RouteTable: RouteTable{
			Identifier: Identifier{
				ID: rtb,
			},
		},
		NATGateway: NATGatewayConfig{
			Preconfigured: true,
		},
	}
}

func NewImportedPrivateSubnet(az string, name string) Subnet {
	return Subnet{
		Identifier: Identifier{
			IDFromStackOutput: name,
		},
		AvailabilityZone: az,
		Private:          true,
	}
}

func NewExistingPublicSubnet(az string, id string) Subnet {
	return Subnet{
		Identifier: Identifier{
			ID: id,
		},
		AvailabilityZone: az,
		Private:          false,
	}
}

func NewImportedPublicSubnet(az string, name string) Subnet {
	return Subnet{
		Identifier: Identifier{
			IDFromStackOutput: name,
		},
		AvailabilityZone: az,
		Private:          false,
	}
}

func (s *Subnet) Provided() bool {
	return s.AvailabilityZone != ""
}

func (s *Subnet) Public() bool {
	return !s.Private
}

func (s *Subnet) AvailabilityZoneLogicalName() string {
	return strings.Replace(strings.Title(s.AvailabilityZone), "-", "", -1)
}

func (s *Subnet) MapPublicIPs() bool {
	return !s.Private
}

func (s *Subnet) ResourcePrefix() string {
	var t string
	if s.Private {
		t = "Private"
	} else {
		t = "Public"
	}
	return t
}

func (s *Subnet) LogicalName() string {
	if s.CustomName != "" {
		return s.CustomName
	}
	return s.ResourcePrefix() + "Subnet" + s.AvailabilityZoneLogicalName()
}

func (s *Subnet) RouteTableID() string {
	return s.RouteTable.ID
}

func (s *Subnet) ManageRouteTable() bool {
	return !s.RouteTable.HasIdentifier()
}

func (s *Subnet) ManageInternetGateway() bool {
	return !s.InternetGateway.HasIdentifier()
}

func (s *Subnet) NATGatewayRouteName() string {
	return s.RouteTableName() + "RouteToNatGateway"
}

// Ref returns ID or ref to newly created resource
func (s *Subnet) Ref() string {
	return s.Identifier.Ref(s.LogicalName())
}

// RouteTableName represents the name of the route table to which this subnet is associated.
func (s *Subnet) RouteTableName() string {
	return s.ResourcePrefix() + "RouteTable" + s.AvailabilityZoneLogicalName()
}

func (s *Subnet) RouteTableRef() string {
	logicalName := s.RouteTableName()
	return s.RouteTable.Ref(logicalName)
}

type InternetGateway struct {
	Identifier `yaml:",inline"`
}

type RouteTable struct {
	Identifier `yaml:",inline"`
}
