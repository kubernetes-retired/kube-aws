package model

import "net"

type EtcdInstance struct {
	IPAddress   net.IP
	SubnetIndex int
}
