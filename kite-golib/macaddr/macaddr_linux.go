package macaddr

import (
	"fmt"
	"net"
)

// Primary gets the hardware address of the primary network interface.
// This is appropriate for use as a machine identifier.
func Primary() (net.HardwareAddr, error) {
	iface, err := net.InterfaceByName("eth0")
	if err != nil {
		return nil, fmt.Errorf("error getting en0 MAC: %v", err)
	}
	return iface.HardwareAddr, nil
}
