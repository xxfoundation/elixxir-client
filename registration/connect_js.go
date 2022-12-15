package registration

import "net"

const defaultURl = "registration.mainnet.cmix.rip"
const defaultIP = "35.157.32.59"

// getAddress returns the correct connection info. For non webassembly,
// it is a simple pass through. For webassembly, it does not
// return the cert
func getConnectionInfo(regAddr, certificate string) (addr string, cert []byte, error) {

	var ip string

	if regAddr == defaultURl {
		ip = defaultIP
	} else {
		var err error
		ip, err = net.LookupIP(regAddr)
		if err != nil {
			return "", nil, err
		}
	}

	_, gwPort, err := net.SplitHostPort(gwAddr)
	if err != nil {
		err = errors.WithMessage(err, "Failed to find port on provided URL")
		return "", nil, err
	}

	addr = ip + ":" + gwPort
	cert = nil

	return addr, cert, nil
}
