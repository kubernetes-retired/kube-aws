package model

type Etcd struct {
	Subnets []Subnet `yaml:"subnets,omitempty"`
}

type EtcdInstance struct {
	Subnet Subnet
}
