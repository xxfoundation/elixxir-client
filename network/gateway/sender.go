////////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Contains gateway message sending wrappers

package gateway

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"io"
)

// Sender Object used for sending that wraps the HostPool for providing destinations
type Sender struct {
	*HostPool
}

// NewSender Create a new Sender object wrapping a HostPool object
func NewSender(poolParams PoolParams, rng io.Reader, ndf *ndf.NetworkDefinition, getter HostManager,
	storage *storage.Session, addGateway chan network.NodeGateway) (*Sender, error) {

	hostPool, err := newHostPool(poolParams, rng, ndf, getter, storage, addGateway)
	if err != nil {
		return nil, err
	}
	return &Sender{hostPool}, nil
}

// SendToSpecific Call given sendFunc to a specific Host in the HostPool,
// attempting with up to numProxies destinations in case of failure
func (m *Sender) SendToSpecific(target *id.ID,
	sendFunc func(host *connect.Host, target *id.ID) (interface{}, error)) (interface{}, error) {
	host, ok := m.getSpecific(target)
	if ok {
		jww.WARN.Printf("SendToSpecific %s", host.GetId().String())
		result, err := sendFunc(host, target)
		if err == nil {
			return result, m.forceAdd([]*id.ID{host.GetId()})
		} else {
			jww.WARN.Printf("Unable to SendToSpecific %s: %+v", host.GetId().String(), err)
		}
	}

	proxies := m.getAny(m.poolParams.ProxyAttempts, []*id.ID{target})
	for proxyIdx := 0; proxyIdx < len(proxies); proxyIdx++ {
		jww.WARN.Printf("SendToSpecific proxy %s", proxies[proxyIdx].GetId().String())
		result, err := sendFunc(proxies[proxyIdx], target)
		if err == nil {
			return result, nil
		} else {
			jww.WARN.Printf("Unable to SendToSpecific proxy %s: %+v", proxies[proxyIdx].GetId().String(), err)
		}
	}

	return nil, errors.Errorf("Unable to send to specific with proxies")
}

// SendToAny Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (m *Sender) SendToAny(sendFunc func(host *connect.Host) (interface{}, error)) (interface{}, error) {

	proxies := m.getAny(m.poolParams.ProxyAttempts, nil)
	for i := range proxies {
		jww.WARN.Printf("Sender %d %s", i, proxies[i].GetId().String())
	}
	for i := range proxies {
		proxy := proxies[i]
		jww.WARN.Printf("SendToAny %s", proxy.GetId().String())
		result, err := sendFunc(proxy)
		if err == nil {
			return result, nil
		} else {
			jww.WARN.Printf("Unable to SendToAny %s: %+v", proxy.GetId().String(), err)
		}
	}

	return nil, errors.Errorf("Unable to send to any proxies")
}

// SendToPreferred Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (m *Sender) SendToPreferred(targets []*id.ID,
	sendFunc func(host *connect.Host, target *id.ID) (interface{}, error)) (interface{}, error) {
	targetHosts := m.getPreferred(targets)
	for i, host := range targetHosts {
		jww.WARN.Printf("SendToPreferred %s", host.GetId().String())
		result, err := sendFunc(host, targets[i])
		if err == nil {
			return result, nil
		} else {
			jww.WARN.Printf("Unable to SendToPreferred %s: %+v", host.GetId().String(), err)
		}
	}

	proxies := m.getAny(m.poolParams.ProxyAttempts, targets)
	for i, proxy := range proxies {
		jww.WARN.Printf("SendToPreferred proxy %s", proxy.GetId().String())
		result, err := sendFunc(proxy, targets[i%len(targets)])
		if err == nil {
			return result, nil
		} else {
			jww.WARN.Printf("Unable to SendToPreferred proxy %s: %+v", proxy.GetId().String(), err)
		}
	}

	return nil, errors.Errorf("Unable to send to any preferred")
}
