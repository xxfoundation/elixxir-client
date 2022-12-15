package gateway

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/authorizer"
	"gitlab.com/xx_network/primitives/id"
	"net"
)

const (
	MaxPoolSize = 7
)

// getAddress returns the correct connection info. For non webassembly,
// it is a simple pass through. For webassembly, it constructs the
// gateway url and returns a nil cert
func getConnectionInfo(gwId *id.ID, gwAddr, certificate string) (addr string, cert []byte, err error) {

	gwUrl := authorizer.GetGatewayDns(gwId.Bytes())
	_, gwPort, err := net.SplitHostPort(gwAddr)
	if err != nil {
		err = errors.WithMessage(err, "Failed to find port on provided URL")
		return "", nil, err
	}
	addr = gwUrl + ":" + gwPort
	cert = nil
	return addr, cert, nil
}
