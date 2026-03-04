package netutil

import "net"

// LocalIP returns the preferred outbound IP address of the machine,
// falling back to "0.0.0.0" if the IP cannot be determined.
func LocalIP() string {
	// 8.8.8.8 is Google's public DNS. Any routable public IP would work here;
	// the OS picks the local interface it would use to reach this destination.
	// No packet is actually sent (UDP is connectionless).
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "0.0.0.0"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
