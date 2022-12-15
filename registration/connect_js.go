package registration

import "net"

const defaultURl = "registration.mainnet.cmix.rip"
const defaultIP = "35.157.32.59"

// getAddress returns the correct connection info. For non webassembly,
// it is a simple pass through. For webassembly, it does not
// return the cert
func getConnectionInfo(regAddr, certificate string) (addr string, cert []byte, err error) {

	var ip string

	regUrl, gwPort, err := net.SplitHostPort(regAddr)
	if err != nil {
		err = errors.WithMessage(err, "Failed to find port on provided URL")
		return "", nil, err
	}

	if regUrl == defaultURl {
		ip = defaultIP
	} else {
		ipObj, err := net.LookupIP(regUrl)
		if err != nil {
			return "", nil, err
		}

		ip = ipObj[0].String()
	}

	addr = ip + ":" + gwPort
	cert = nil

	return addr, cert, nil
}
