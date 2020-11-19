package ud

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
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
	rng     *fastRNG.StreamGenerator
	grp     *cyclic.Group
	sw      interfaces.Switchboard
	storage *storage.Session

	udID *id.ID

	inProgressLookup map[uint64]chan *LookupResponse
	inProgressMux    sync.RWMutex

	net interfaces.NetworkManager

	commID     uint64
	commIDLock sync.Mutex
}

func (m *Manager) StartProcesses() stoppable.Stoppable {

	lookupStop := stoppable.NewSingle("UDLookup")
	lookupChan := make(chan message.Receive, 100)
	m.sw.RegisterChannel("UDLookupResponse", m.udID, message.UdLookupResponse, lookupChan)
	go m.lookupProcess(lookupChan, lookupStop.Quit())

	udMulti := stoppable.NewMulti("UD")
	udMulti.Add(lookupStop)
	return lookupStop
}
