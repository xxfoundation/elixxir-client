package ud

import (
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

type Manager struct {
	comms   *client.Comms
	host    *connect.Host
	privKey *rsa.PrivateKey
}

