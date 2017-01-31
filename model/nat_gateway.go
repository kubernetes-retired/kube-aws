package model

import (
	"fmt"
)

type NATGatewayConfig struct {
	Identifier      `yaml:",inline"`
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
	NATGatewayRouteName() (string, error)
	Ref() string
	PrivateSubnetRouteTableRef() (string, error)
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
	return g.privateSubnet.ManageNATGateway()
}

func (g natGatewayImpl) ManageEIP() bool {
	return g.EIPAllocationID == ""
}

func (g natGatewayImpl) ManageRoute() bool {
	return g.privateSubnet.ManageRouteToNATGateway()
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

func (g natGatewayImpl) PrivateSubnetRouteTableRef() (string, error) {
	ref, err := g.privateSubnet.RouteTableRef()
	if err != nil {
		return "", err
	}
	return ref, nil
}

func (g natGatewayImpl) NATGatewayRouteName() (string, error) {
	return fmt.Sprintf("%sRouteToNatGateway", g.privateSubnet.ReferenceName()), nil
}

func (g natGatewayImpl) Validate() error {
	if !g.ManageNATGateway() {
		if !g.privateSubnet.HasIdentifier() {
			return fmt.Errorf("a preconfigured NGW must be associated to an existing private subnet: %+v", g)
		}

		if g.publicSubnet.Provided() {
			return fmt.Errorf("a preconfigured NGW must not be associated to an existing public subnet: %+v", g)
		}

		if !g.privateSubnet.RouteTable.HasIdentifier() {
			return fmt.Errorf("a preconfigured NGW must have an existing route table provided via routeTable.id or routeTable.idFromStackOutput: %+v", g)
		}

		if g.HasIdentifier() {
			return fmt.Errorf("a preconfigured NGW must not have id or idFromStackOutput: %+v", g)
		}

		if g.EIPAllocationID != "" {
			return fmt.Errorf("a preconfigured NGW must not have an eipAllocactionID: %+v", g)
		}
	}
	return nil
}
