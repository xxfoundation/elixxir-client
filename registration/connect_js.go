package registration

import "strings"

const toReplace = "registration"
const replaceWith = "registrar"

// getAddress returns the correct connection info. For non webassembly,
// it is a simple pass through. For webassembly, it does not
// return the cert
func getConnectionInfo(regAddr, certificate string) (addr string, cert []byte, err error) {
	addr = strings.Replace(regAddr, toReplace, replaceWith, 1)

	cert = []byte(certificate)

	return addr, cert, nil
}
