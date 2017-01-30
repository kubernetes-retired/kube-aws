package model

import "fmt"

type Etcd struct {
	Subnets []Subnet `yaml:"subnets,omitempty"`
}

type EtcdInstance interface {
	SubnetRef() string
	DependencyRef() string
}

type etcdInstanceImpl struct {
	subnet     Subnet
	natGateway NATGateway
}

func NewPrivateEtcdInstance(s Subnet, ngw NATGateway) EtcdInstance {
	return etcdInstanceImpl{
		subnet:     s,
		natGateway: ngw,
	}
}

func NewPublicEtcdInstance(s Subnet) EtcdInstance {
	return etcdInstanceImpl{
		subnet: s,
	}
}

func (i etcdInstanceImpl) SubnetRef() string {
	return i.subnet.Ref()
}

func (i etcdInstanceImpl) DependencyRef() string {
	// We have to wait until the route to the NAT gateway if it doesn't exist yet(hence ManageRoute=true) or the etcd node fails due to inability to connect internet
	if i.subnet.Private && i.natGateway.ManageRoute() {
		return fmt.Sprintf(`"%s"`, i.natGateway.NATGatewayRouteName())
	}
	return ""
}
