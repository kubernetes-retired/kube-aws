package model

import (
	"fmt"
)

type NATGatewayConfig struct {
	Identifier      `yaml:",inline"`
	EIPAllocationID string `yaml:"eipAllocationId,omitempty"`
}

func (c NATGatewayConfig) Validate() error {
	if c.HasIdentifier() && c.EIPAllocationID != "" {
		return fmt.Errorf("eipAllocationId can't be specified for a existing nat gatway. It is an user's responsibility to configure the nat gateway if one tried to reuse an existing one: %+v", c)
	}
	return nil
}

// kube-aws manages at most one NAT gateway per subnet
type NATGateway interface {
	EIPAllocationIDRef() string
	EIPLogicalName() string
	IsConnectedToPrivateSubnet(Subnet) bool
	LogicalName() string
	ManageEIP() bool
	ManageNATGateway() bool
	ManageRoute() bool
	Ref() string
	PublicSubnetRef() string
	PrivateSubnets() []Subnet
	Validate() error
}

type natGatewayImpl struct {
	NATGatewayConfig
	privateSubnets []Subnet
	publicSubnet   Subnet
}

func NewNATGateway(c NATGatewayConfig, private Subnet, public Subnet) NATGateway {
	return natGatewayImpl{
		NATGatewayConfig: c,
		privateSubnets:   []Subnet{private},
		publicSubnet:     public,
	}
}

func (g natGatewayImpl) LogicalName() string {
	name := ""
	for _, s := range g.privateSubnets {
		name = name + s.LogicalName()
	}
	return fmt.Sprintf("NatGateway%s", name)
}

func (g natGatewayImpl) ManageNATGateway() bool {
	allTrue := true
	allFalse := true
	for _, s := range g.privateSubnets {
		allTrue = allTrue && s.ManageNATGateway()
		allFalse = allFalse && !s.ManageNATGateway()
	}
	if allTrue {
		return true
	} else if allFalse {
		return false
	}

	panic(fmt.Sprintf("[bug] assertion failed: private subnets associated to this nat gateway(%+v) conflicts in their settings. kube-aws is confused and can't decide whether it should manage the nat gateway or not", g))
}

func (g natGatewayImpl) ManageEIP() bool {
	return g.EIPAllocationID == ""
}

func (g natGatewayImpl) ManageRoute() bool {
	allTrue := true
	allFalse := true
	for _, s := range g.privateSubnets {
		allTrue = allTrue && s.ManageRouteToNATGateway()
		allFalse = allFalse && !s.ManageRouteToNATGateway()
	}
	if allTrue {
		return true
	} else if allFalse {
		return false
	}

	panic(fmt.Sprintf("[bug] assertion failed: private subnets associated to this nat gateway(%+v) conflicts in their settings. kube-aws is confused and can't decide whether it should manage the route to nat gateway or not", g))
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
	for _, ps := range g.privateSubnets {
		if ps.LogicalName() == s.LogicalName() {
			return true
		}
	}
	return false
}

func (g natGatewayImpl) Ref() string {
	return g.Identifier.Ref(g.LogicalName)
}

func (g natGatewayImpl) PublicSubnetRef() string {
	return g.publicSubnet.Ref()
}

func (g natGatewayImpl) PrivateSubnets() []Subnet {
	return g.privateSubnets
}

func (g natGatewayImpl) Validate() error {
	if err := g.NATGatewayConfig.Validate(); err != nil {
		return fmt.Errorf("failed to validate nat gateway: %v", err)
	}
	if !g.ManageNATGateway() {
		for i, s := range g.privateSubnets {
			if !s.HasIdentifier() {
				return fmt.Errorf("a preconfigured NGW must be associated to an existing private subnet #%d: %+v", i, g)
			}

			if !s.RouteTable.HasIdentifier() {
				return fmt.Errorf("a preconfigured NGW must have an existing route table provided via routeTable.id or routeTable.idFromStackOutput: %+v", g)
			}
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
