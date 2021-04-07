////////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Contains gateway message sending wrappers

package gateway

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"io"
)

// Object used for sending that wraps the HostPool for providing destinations
type Sender struct {
	*HostPool
}

// Create a new Sender object wrapping a HostPool object
func NewSender(poolParams PoolParams, rng io.Reader, ndf *ndf.NetworkDefinition, getter HostManager,
	storage *storage.Session, addGateway chan network.NodeGateway) (*Sender, error) {

	hostPool, err := newHostPool(poolParams, rng, ndf, getter, storage, addGateway)
	if err != nil {
		return nil, err
	}
	return &Sender{hostPool}, nil
}

// Call given sendFunc to a specific Host in the HostPool,
// attempting with up to numProxies destinations in case of failure
func (m *Sender) SendToSpecific(targets []*id.ID,
	sendFunc func(host *connect.Host, target *id.ID) (interface{}, error)) (interface{}, error) {
	for _, target := range targets {
		host, ok := m.GetSpecific(target)

		if ok {
			result, err := sendFunc(host, target)
			if err == nil {
				return result, m.ForceAdd([]*id.ID{host.GetId()})
			}
		}
	}

	return m.SendToAny(sendFunc)
}

// Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (m *Sender) SendToAny(sendFunc func(host *connect.Host) (interface{}, error)) (interface{}, error) {

	proxies := m.GetAny(m.poolParams.ProxyAttempts)
	for _, proxy := range proxies {
		result, err := sendFunc(proxy)
		if err == nil {
			return result, nil
		}
	}

	return nil, errors.Errorf("Unable to send to any proxies")
}

// Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (m *Sender) SendToPreferred(targets []*id.ID,
	sendFunc func(host *connect.Host, target *id.ID) (interface{}, error)) (interface{}, error) {

	targetHosts := m.GetPreferred(targets)
	for i, host := range targetHosts {
		result, err := sendFunc(host, targets[i])
		if err == nil {
			return result, nil
		}
	}

	return nil, errors.Errorf("Unable to send to any preferred")
}
