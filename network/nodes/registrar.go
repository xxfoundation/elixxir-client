package nodes

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"sync"
	"time"
)

const InputChanLen = 1000
const maxAttempts = 5

var delayTable = [5]time.Duration{
	0,
	5 * time.Second,
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
}

type Registrar interface {
	StartProcesses(numParallel uint) stoppable.Stoppable
	Has(nid *id.ID) bool
	Remove(nid *id.ID)
	GetKeys(topology *connect.Circuit) (MixCypher, error)
	NumRegistered() int
	GetInputChannel() chan<- network.NodeGateway
	TriggerRegistration(nid *id.ID)
}

type RegisterNodeCommsInterface interface {
	SendRequestClientKeyMessage(host *connect.Host,
		message *pb.SignedClientKeyRequest) (*pb.SignedKeyResponse, error)
}

type registrar struct {
	nodes map[id.ID]*key
	kv    *versioned.KV
	mux   sync.RWMutex

	session *storage.Session
	sender  *gateway.Sender
	comms   RegisterNodeCommsInterface
	rng     *fastRNG.StreamGenerator

	c chan network.NodeGateway
}

// LoadRegistrar loads a registrar from disk, and creates a new one if it does
// not exist.
func LoadRegistrar(kv *versioned.KV, session *storage.Session,
	sender *gateway.Sender, comms RegisterNodeCommsInterface,
	rngGen *fastRNG.StreamGenerator) (Registrar, error) {
	kv = kv.Prefix(prefix)
	r := &registrar{
		nodes: make(map[id.ID]*key),
		kv:    kv,
	}

	obj, err := kv.Get(storeKey, currentKeyVersion)
	// If there is no stored data, make a new node handler
	if err != nil {
		jww.WARN.Printf("Failed to load Node Registrar, creating a new object")
		err = r.save()
		if err != nil {
			return nil, errors.WithMessagef(err, "Failed to make a new registrar")
		}
	} else {
		err = r.unmarshal(obj.Data)
		if err != nil {
			return nil, err
		}
	}

	r.session = session
	r.sender = sender
	r.comms = comms
	r.rng = rngGen

	r.c = make(chan network.NodeGateway, InputChanLen)

	return r, nil
}

func (r *registrar) StartProcesses(numParallel uint) stoppable.Stoppable {
	multi := stoppable.NewMulti("NodeRegistrations")

	inProgress := &sync.Map{}
	// We are relying on the in progress check to ensure there is only a single
	// operator at a time, as a result this is a map of ID -> int
	attempts := &sync.Map{}

	for i := uint(0); i < numParallel; i++ {
		stop := stoppable.NewSingle(fmt.Sprintf("NodeRegistration %d", i))

		go registerNodes(r, stop, inProgress, attempts)
		multi.Add(stop)
	}

	return multi
}

func (r *registrar) GetInputChannel() chan<- network.NodeGateway {
	return r.c
}
func (r *registrar) TriggerRegistration(nid *id.ID) {
	r.c <- network.NodeGateway{
		Node: ndf.Node{ID: nid.Marshal(),
			//status must be active because it is in a round
			Status: ndf.Active},
	}
}

// GetKeys returns a MixCypher for the topology and a list of nodes it did
// not have a key for. If there are missing keys, then returns nil MixCypher.
func (r *registrar) GetKeys(topology *connect.Circuit) (MixCypher, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	keys := make([]*key, topology.Len())

	// Get keys for every node. If it cannot be found, thn add it to the missing
	// nodes list so that it can be
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		k, ok := r.nodes[*nid]
		if !ok {
			r.c <- network.NodeGateway{
				Node: ndf.Node{
					ID:     nid.Marshal(),
					Status: ndf.Active, // Status must be active because it is in a round
				},
			}
			return nil, errors.Errorf(
				"cannot get key for %s, triggered registration", nid)
		} else {
			keys[i] = k
		}
	}

	rk := &mixCypher{
		keys: keys,
		g:    r.session.GetCmixGroup(),
	}

	return rk, nil
}

// Has returns if the store has the nodes.
func (r *registrar) Has(nid *id.ID) bool {
	r.mux.RLock()
	_, exists := r.nodes[*nid]
	r.mux.RUnlock()
	return exists
}

func (r *registrar) Remove(nid *id.ID) {
	r.remove(nid)
}

// NumRegistered returns the number of registered nodes.
func (r *registrar) NumRegistered() int {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return len(r.nodes)
}
