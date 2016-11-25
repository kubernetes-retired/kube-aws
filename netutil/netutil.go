package netutil

import "net"

//Does the address space of these networks "a" and "b" overlap?
func CidrOverlap(a, b *net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP)
}

//Return next IP address in network range
func IncrementIP(netIP net.IP) net.IP {
	ip := make(net.IP, len(netIP))
	copy(ip, netIP)

	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}

	return ip
}
