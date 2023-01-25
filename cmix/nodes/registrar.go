////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/gateway"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const InputChanLen = 1000
const maxAttempts = 5

// Backoff for attempting to register with a cMix node.
var delayTable = [5]time.Duration{
	0,
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
	240 * time.Second,
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

	inProgress sync.Map
	// We are relying on the in progress check to ensure there is only a single
	// operator at a time, as a result this is a map of ID -> int
	attempts sync.Map

	pauser        chan interface{}
	resumer       chan interface{}
	numberRunning *int64
	maxRunning    int

	runnerLock sync.Mutex

	numnodesGetter func() int

	c chan network.NodeGateway
}

// LoadRegistrar loads a Registrar from disk or creates a new one if it does not
// exist.
func LoadRegistrar(session session, sender gateway.Sender,
	comms RegisterNodeCommsInterface, rngGen *fastRNG.StreamGenerator,
	c chan network.NodeGateway, numNodesGetter func() int) (Registrar, error) {

	running := int64(0)

	kv := session.GetKV().Prefix(prefix)
	r := &registrar{
		nodes:          make(map[id.ID]*key),
		kv:             kv,
		pauser:         make(chan interface{}),
		resumer:        make(chan interface{}),
		numberRunning:  &running,
		numnodesGetter: numNodesGetter,
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
	r.runnerLock.Lock()
	defer r.runnerLock.Unlock()

	multi := stoppable.NewMulti("NodeRegistrations")
	r.maxRunning = int(numParallel)

	for i := uint(0); i < numParallel; i++ {
		stop := stoppable.NewSingle("NodeRegistration " + strconv.Itoa(int(i)))

		go registerNodes(r, r.session, stop, &r.inProgress, &r.attempts, int(i))
		multi.Add(stop)
	}

	return multi
}

// PauseNodeRegistrations stops all node registrations
// and returns a function to resume them
func (r *registrar) PauseNodeRegistrations(timeout time.Duration) error {
	r.runnerLock.Lock()
	defer r.runnerLock.Unlock()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	numRegistrations := atomic.LoadInt64(r.numberRunning)
	jww.INFO.Printf("PauseNodeRegistrations() - Pausing %d registrations", numRegistrations)
	for i := int64(0); i < numRegistrations; i++ {
		select {
		case r.pauser <- struct{}{}:
		case <-timer.C:
			return errors.Errorf("Timed out on pausing node registration on %d", i)
		}
	}

	return nil
}

// ChangeNumberOfNodeRegistrations changes the number of parallel node
// registrations up to the initialized maximum
func (r *registrar) ChangeNumberOfNodeRegistrations(toRun int,
	timeout time.Duration) error {
	r.runnerLock.Lock()
	defer r.runnerLock.Unlock()
	numRunning := int(atomic.LoadInt64(r.numberRunning))
	if toRun+numRunning > r.maxRunning {
		return errors.Errorf("Cannot change number of " +
			"running node registration to number greater than the max")
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	if numRunning < toRun {
		jww.INFO.Printf("ChangeNumberOfNodeRegistrations(%d) Reducing number "+
			"of node registrations from %d to %d", toRun, numRunning, toRun)
		for i := 0; i < toRun-numRunning; i++ {
			select {
			case r.pauser <- struct{}{}:
			case <-timer.C:
				return errors.New("Timed out on reducing node registration")
			}
		}
	} else if numRunning > toRun {
		jww.INFO.Printf("ChangeNumberOfNodeRegistrations(%d) Increasing number "+
			"of node registrations from %d to %d", toRun, numRunning, toRun)
		for i := 0; i < toRun-numRunning; i++ {
			select {
			case r.resumer <- struct{}{}:
			case <-timer.C:
				return errors.New("Timed out on increasing node registration")
			}
		}
	}
	return nil
}

// GetNodeKeys returns a MixCypher for the topology and a list of nodes it did
// not have a key for. If there are missing keys, then returns nil.
func (r *registrar) GetNodeKeys(topology *connect.Circuit) (MixCypher, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	keys := make([]*key, topology.Len())
	ephemeralKeys := make([]bool, topology.Len())
	missingNodes := make(map[id.ID]int)

	// Get keys for every node. If it cannot be found, then add it to the
	// missing nodes list so that it can be.
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		k, ok := r.nodes[*nid]
		if !ok {
			gwID := nid.DeepCopy()
			gwID.SetType(id.Gateway)
			r.c <- network.NodeGateway{
				Node: ndf.Node{
					ID:     nid.Marshal(),
					Status: ndf.Active, // Must be active because it is in a round
				},
				Gateway: ndf.Gateway{
					ID: gwID.Marshal(),
				},
			}

			jww.WARN.Println(errors.Errorf(
				"cannot get key for %s, triggered registration", nid))
			missingNodes[*nid] = i
			ephemeralKeys[i] = true
		} else {
			keys[i] = k
		}
	}

	// Cannot attempt to send without at least one registered node
	if len(missingNodes) == topology.Len() {
		return nil, errors.New("Must have at least one registered node to create mixCypher")
	}
	var edPub []byte
	if len(missingNodes) > 0 {
		// Generate temp ed25519 for this send
		priv, pub := ecdh.ECDHNIKE.NewKeypair(r.rng.GetStream())
		jww.INFO.Printf("Generated ephemeral keypair for sending to unregistered nodes")
		edPub = pub.Bytes()
		currentNdf := r.session.GetNDF()
		for nid, keyIndex := range missingNodes {
			for _, n := range currentNdf.Nodes { // TODO: any more efficient way to get the corresponding keys?
				if bytes.Compare(n.ID, nid[:]) == 0 {
					nodePubKey, err := ecdh.ECDHNIKE.UnmarshalBinaryPublicKey(n.Ed25519)
					if err != nil {
						return nil, errors.WithMessagef(err, "Failed to unmarshal binary pubkey %+v", n.Ed25519)
					}
					secret := priv.DeriveSecret(nodePubKey)
					nodeSecretHash := hash.CMixHash.New()
					nodeSecretHash.Reset()
					nodeSecretHash.Write(r.session.GetTransmissionID().Bytes())
					nodeSecretHash.Write(secret)
					hashBytes := nodeSecretHash.Sum(nil)
					k := r.session.GetCmixGroup().NewIntFromBytes(hashBytes)
					keys[keyIndex] = &key{nil, k, nil, uint64(time.Now().Add(time.Second * 5).UnixNano()), ""} //newKey(r.kv, k, &nid, uint64(time.Now().Add(time.Second*5).UnixNano()), nil)
					break
				}
			}
		}
	}

	rk := &mixCypher{
		keys:          keys,
		g:             r.session.GetCmixGroup(),
		ephemeralKeys: ephemeralKeys,
		ephemeralKey:  edPub,
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
