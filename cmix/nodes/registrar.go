package nodes

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"strconv"
	"sync"
	"time"
)

const InputChanLen = 1000
const maxAttempts = 5

// Backoff for attempting to register with a cMix node.
var delayTable = [5]time.Duration{
	0,
	5 * time.Second,
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
}

// registrar is an implementation of the Registrar interface.
type registrar struct {
	nodes map[id.ID]*key
	kv    *versioned.KV
	mux   sync.RWMutex

	session session
	sender  gateway.Sender
	comms   RegisterNodeCommsInterface
	rng     *fastRNG.StreamGenerator

	c chan network.NodeGateway
}

// LoadRegistrar loads a Registrar from disk or creates a new one if it does not
// exist.
func LoadRegistrar(session session, sender gateway.Sender,
	comms RegisterNodeCommsInterface, rngGen *fastRNG.StreamGenerator,
	c chan network.NodeGateway) (Registrar, error) {

	kv := session.GetKV().Prefix(prefix)
	r := &registrar{
		nodes: make(map[id.ID]*key),
		kv:    kv,
	}

	obj, err := kv.Get(storeKey, currentKeyVersion)
	if err != nil {
		// If there is no stored data, make a new node handler
		jww.WARN.Printf("Failed to load Node Registrar, creating a new object.")
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

	r.c = c

	return r, nil
}

// StartProcesses initiates numParallel amount of threads
// to register with nodes.
func (r *registrar) StartProcesses(numParallel uint) stoppable.Stoppable {
	multi := stoppable.NewMulti("NodeRegistrations")

	inProgress := &sync.Map{}

	// We are relying on the in progress check to ensure there is only a single
	// operator at a time, as a result this is a map of ID -> int
	attempts := &sync.Map{}

	for i := uint(0); i < numParallel; i++ {
		stop := stoppable.NewSingle("NodeRegistration " + strconv.Itoa(int(i)))

		go registerNodes(r, r.session, stop, inProgress, attempts)
		multi.Add(stop)
	}

	return multi
}

// GetNodeKeys returns a MixCypher for the topology and a list of nodes it did
// not have a key for. If there are missing keys, then returns nil.
func (r *registrar) GetNodeKeys(topology *connect.Circuit) (MixCypher, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	keys := make([]*key, topology.Len())

	// Get keys for every node. If it cannot be found, then add it to the
	// missing nodes list so that it can be.
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		k, ok := r.nodes[*nid]
		if !ok {
			r.c <- network.NodeGateway{
				Node: ndf.Node{
					ID:     nid.Marshal(),
					Status: ndf.Active, // Must be active because it is in a round
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

// HasNode returns true if the registrar has the node.
func (r *registrar) HasNode(nid *id.ID) bool {
	r.mux.RLock()
	defer r.mux.RUnlock()

	_, exists := r.nodes[*nid]

	return exists
}

// RemoveNode removes the node from the registrar.
func (r *registrar) RemoveNode(nid *id.ID) {
	r.remove(nid)
}

// NumRegisteredNodes returns the number of registered nodes.
func (r *registrar) NumRegisteredNodes() int {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return len(r.nodes)
}

// GetInputChannel returns the send-only channel for registering with
// a cMix node.
func (r *registrar) GetInputChannel() chan<- network.NodeGateway {
	return r.c
}

// TriggerNodeRegistration initiates a registration with the given
// cMix node by sending on the registrar's registration channel.
func (r *registrar) TriggerNodeRegistration(nid *id.ID) {
	r.c <- network.NodeGateway{
		Node: ndf.Node{
			ID:     nid.Marshal(),
			Status: ndf.Active, // Must be active because it is in a round
		},
	}
}
