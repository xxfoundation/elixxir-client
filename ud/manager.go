package ud

import (
	"encoding/binary"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/cyclic"
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
	grp *cyclic.Group
	sw interfaces.Switchboard

	udID *id.ID


	inProgressLookup map[uint64]chan *LookupResponse
	inProgressMux sync.RWMutex

	net interfaces.NetworkManager
}

func (m *Manager)getCommID()(uint64, error){
	//fixme: this should use incremenetation
	stream := m.rng.GetStream()

	idBytes := make([]byte, 8)
	if _, err := stream.Read(idBytes); err!=nil{
		return 0, err
	}

	return binary.BigEndian.Uint64(idBytes), nil
}

func (m *Manager)StartProcessies()stoppable.Stoppable{

	lookupStop := stoppable.NewSingle("UDLookup")
	lookupChan := make(chan message.Receive, 100)
	m.sw.RegisterChannel("UDLookupResponse", m.udID, message.UdLookupResponse, lookupChan)
	go m.lookupProcess(lookupChan, lookupStop.Quit())


	udMulti := stoppable.NewMulti("UD")
	udMulti.Add(lookupStop)
	return lookupStop
}