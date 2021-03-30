////////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Contains gateway message sending wrappers

package gateway

import (
	"github.com/pkg/errors"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

// Object used for sending that wraps the HostPool for providing destinations
type Mesh struct {
	HostPool
}

// Call given sendFunc to a specific Host in the HostPool,
// attempting with up to numProxies destinations in case of failure
func (m *Mesh) SendToSpecific(target *id.ID, numProxies int,
	sendFunc func(host *connect.Host) (interface{}, error)) (interface{}, error) {
	host, ok := m.GetSpecific(target)

	if ok {
		result, err := sendFunc(host)
		if err == nil {
			return result, m.ForceAdd([]*id.ID{host.GetId()})
		}
	}

	return m.SendToAny(numProxies, sendFunc)
}

// Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (m *Mesh) SendToAny(numProxies int,
	sendFunc func(host *connect.Host) (interface{}, error)) (interface{}, error) {

	proxies := m.GetAny(numProxies)
	for _, proxy := range proxies {
		result, err := sendFunc(proxy)
		if err == nil {
			return result, nil
		}
	}

	return nil, errors.Errorf("Unable to send to any proxies")
}
