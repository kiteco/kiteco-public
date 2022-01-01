package macaddr

import (
	"fmt"
	"net"
)

// Primary gets the hardware address of the primary network interface.
// This is appropriate for use as a machine identifier.
func Primary() (net.HardwareAddr, error) {
	// Set machine ID to ethernet mac address.
	// Hash the mac address using md5 hash and hex encode it.
	iface, err := net.InterfaceByName("Local Area Connection")
	if err != nil {
		return nil, fmt.Errorf("error getting MAC: %v", err)
	}
	return iface.HardwareAddr, nil
}
