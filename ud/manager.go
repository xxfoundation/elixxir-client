package ud

import (
	"encoding/binary"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type Manager struct {
	comms   *client.Comms
	host    *connect.Host
	privKey *rsa.PrivateKey
	rng *fastRNG.StreamGenerator

	udID *id.ID


	inProgressLookup map[int64]chan *LookupResponse
	inProgressMux sync.RWMutex

	net interfaces.NetworkManager
}

func (m *Manager)getCommID()(uint64, error){
	stream := m.rng.GetStream()

	idBytes := make([]byte, 8)
	if _, err := stream.Read(idBytes); err!=nil{
		return 0, err
	}

	return binary.BigEndian.Uint64(idBytes), nil
}