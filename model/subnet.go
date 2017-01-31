package model

import (
	"fmt"
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

func NewPublicSubnetWithPreconfiguredInternetGateway(az string, cidr string, rtb string) Subnet {
	return Subnet{
		AvailabilityZone: az,
		InstanceCIDR:     cidr,
		Private:          false,
		RouteTable: RouteTable{
			Identifier: Identifier{
				ID: rtb,
			},
		},
		InternetGateway: InternetGateway{},
	}
}

func NewPrivateSubnetWithPreconfiguredNATGateway(az string, cidr string, rtb string) Subnet {
	return Subnet{
		AvailabilityZone: az,
		InstanceCIDR:     cidr,
		Private:          true,
		RouteTable: RouteTable{
			Identifier: Identifier{
				ID: rtb,
			},
		},
		NATGateway: NATGatewayConfig{},
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

func (s *Subnet) ReferenceName() string {
	if s.ManageSubnet() {
		return s.LogicalName()
	} else if s.ID != "" {
		return s.ID
	}
	return s.IDFromStackOutput
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

// ManageNATGateway returns true if a NAT gateway for this subnet must be created or updated by kube-aws
// kube-aws creates or updates a NAT gateway if:
// * the subnet is private and
// * the subnet is going to be managed by kube-aws(an existing subnet is NOT specified) and
// * the route table for the subnet is going to be managed by kube-aws(an existing subnet is NOT specified) and
// * an existing NAT gateway ID is not specified to be reused
func (s *Subnet) ManageNATGateway() bool {
	return s.Private && s.ManageRouteTable() && !s.NATGateway.HasIdentifier()
}

// ManageRouteToNATGateway returns true if a route to a NAT gateway for this subnet must be created or updated by kube-aws
// kube-aws creates or updates a NAT gateway if:
// * the NGW is going to be managed or
// * an existing NAT gateway ID is specified
func (s *Subnet) ManageRouteToNATGateway() bool {
	return s.ManageNATGateway() || s.NATGateway.HasIdentifier()
}

// ManageRouteTable returns true if a route table for this subnet must be created or updated by kube-aws
// kube-aws creates a route table if and only if the subnet is also going to be managed and an existing route table for it isn't specified
func (s *Subnet) ManageRouteTable() bool {
	return s.ManageSubnet() && !s.RouteTable.HasIdentifier()
}

// ManageRouteToInternet returns true if a route from this subnet to to an IGW must be created or updated by kube-aws
// kube-aws creates a route to an IGW for an subnet if and only if:
// * the subnet is public and
// * the subnet is going to be managed by kube-aws and
// * the route table is going to be managed by kube-aws
// In other words, kube-aws won't create or update a route to an IGW if:
// * the subnet is private or
// * an existing subnet is used or
// * an existing route table is used
func (s *Subnet) ManageRouteToInternet() bool {
	return s.Public() && s.ManageSubnet() && s.ManageRouteTable()
}

// ManageSubnet returns true if this subnet must be managed(created or updated) by kube-aws
// kube-aws creates a subnet if subnet.id and subnet.idFromStackOutput are not specified
func (s *Subnet) ManageSubnet() bool {
	return !s.HasIdentifier()
}

// Ref returns ID or ref to newly created resource
func (s *Subnet) Ref() string {
	return s.Identifier.Ref(s.LogicalName())
}

// RouteTableName represents the name of the route table to which this subnet is associated.
func (s *Subnet) RouteTableName() (string, error) {
	// There should be no need to call this func if the route table isn't going to be created/updated by kube-aws
	if !s.ManageRouteTable() {
		return "", fmt.Errorf("[bug] assertion failed: RouteTableName() must be called if and only if ManageRouteTable() returns true")
	}
	return s.ResourcePrefix() + "RouteTable" + s.AvailabilityZoneLogicalName(), nil
}

func (s *Subnet) RouteTableRef() (string, error) {
	return s.RouteTable.IdOrRef(s.RouteTableName)
}

type RouteTable struct {
	Identifier `yaml:",inline"`
}
