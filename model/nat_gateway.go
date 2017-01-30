package model

import (
	"fmt"
)

type NATGatewayConfig struct {
	Identifier      `yaml:",inline"`
	Preconfigured   bool   `yaml:"preconfigured,omitempty"`
	EIPAllocationID string `yaml:"eipAllocationId,omitempty"`
}

type NATGateway interface {
	EIPAllocationIDRef() string
	EIPLogicalName() string
	IsConnectedToPrivateSubnet(Subnet) bool
	LogicalName() string
	ManageEIP() bool
	ManageNATGateway() bool
	ManageRoute() bool
	NATGatewayRouteName() string
	Ref() string
	PrivateSubnetRouteTableRef() string
	PublicSubnetRef() string
	Validate() error
}

type natGatewayImpl struct {
	NATGatewayConfig
	privateSubnet Subnet
	publicSubnet  Subnet
}

func NewNATGateway(c NATGatewayConfig, private Subnet, public Subnet) NATGateway {
	return natGatewayImpl{
		NATGatewayConfig: c,
		privateSubnet:    private,
		publicSubnet:     public,
	}
}

func (g natGatewayImpl) LogicalName() string {
	return fmt.Sprintf("NatGateway%s", g.privateSubnet.AvailabilityZoneLogicalName())
}

func (g natGatewayImpl) ManageNATGateway() bool {
	return !g.HasIdentifier() && !g.Preconfigured
}

func (g natGatewayImpl) ManageEIP() bool {
	return g.EIPAllocationID == ""
}

func (g natGatewayImpl) ManageRoute() bool {
	return !g.Preconfigured
}

func (g natGatewayImpl) EIPLogicalName() string {
	return fmt.Sprintf("%sEIP", g.LogicalName())
}

func (g natGatewayImpl) EIPAllocationIDRef() string {
	if g.ManageEIP() {
		return fmt.Sprintf(`{"Fn::GetAtt": ["%s", "AllocationId"]}`, g.EIPLogicalName())
	}
	return g.EIPAllocationID
}

func (g natGatewayImpl) IsConnectedToPrivateSubnet(s Subnet) bool {
	return g.privateSubnet.LogicalName() == s.LogicalName()
}

func (g natGatewayImpl) Ref() string {
	return g.Identifier.Ref(g.LogicalName())
}

func (g natGatewayImpl) PublicSubnetRef() string {
	return g.publicSubnet.Ref()
}

func (g natGatewayImpl) PrivateSubnetRouteTableRef() string {
	return g.privateSubnet.RouteTableRef()
}

func (g natGatewayImpl) NATGatewayRouteName() string {
	return g.privateSubnet.NATGatewayRouteName()
}

func (g natGatewayImpl) Validate() error {
	if g.Preconfigured {
		if !g.privateSubnet.HasIdentifier() {
			return fmt.Errorf("an NGW with preconfigured=true must be associated to an existing private subnet: %+v", g)
		}

		if g.publicSubnet.Provided() {
			return fmt.Errorf("an NGW with preconfigured=true must not be associated to an existing public subnet: %+v", g)
		}

		if !g.privateSubnet.RouteTable.HasIdentifier() {
			return fmt.Errorf("an NGW with preconfigured=true must have an existing route table provided via routeTable.id or routeTable.idFromStackOutput: %+v", g)
		}

		if g.HasIdentifier() {
			return fmt.Errorf("an NGW with preconcfigured=true must not have id or idFromStackOutput: %+v", g)
		}

		if g.EIPAllocationID != "" {
			return fmt.Errorf("an NGW with preconcfigured=true must not have an eipAllocactionID: %+v", g)
		}
	}
	return nil
}
