package network

import (
	"fmt"
	"net"
)

type IPBinding string

const (
	LocalHostBinding     IPBinding = "127.0.0.1"
	AllInterfacesBinding IPBinding = "0.0.0.0"
)

// ResolveAddr resolves the passed in TCP address
// The second param is the default ip to bind to, if no ip address is specified
func ResolveAddr(address string, defaultIP IPBinding) (*net.TCPAddr, error) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addr '%s': %w", address, err)
	}

	if addr.IP == nil {
		addr.IP = net.ParseIP(string(defaultIP))
	}

	return addr, nil
}
