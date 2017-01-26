package model

import "fmt"

type NATGatewayConfig struct {
	Identifier      `yaml:",inline"`
	EIPAllocationID string `yaml:"eipAllocationId,omitempty"`
}

type NATGateway interface {
	LogicalName() string
	ManageNATGateway() bool
	ManageEIP() bool
	EIPLogicalName() string
	EIPAllocationIDRef() string
	Ref() string
	PublicSubnetRef() string
	PrivateSubnetRouteTableRef() string
	NATGatewayRouteName() string
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
	return !g.HasIdentifier()
}

func (g natGatewayImpl) ManageEIP() bool {
	return g.EIPAllocationID == ""
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
