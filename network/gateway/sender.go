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
func (s *Sender) SendToSpecific(target *id.ID,
	sendFunc func(host *connect.Host, target *id.ID) (interface{}, bool, error)) (interface{}, error) {
	host, ok := s.getSpecific(target)
	if ok {
		result, didAbort, err := sendFunc(host, target)
		if err == nil {
			return result, s.forceAdd([]*id.ID{host.GetId()})
		} else {
			if didAbort {
				return nil, errors.WithMessagef(err, "Aborted SendToSpecific gateway %s", host.GetId().String())
			}
			jww.WARN.Printf("Unable to SendToSpecific %s: %+v", host.GetId().String(), err)
		}
	}

	proxies := s.getAny(s.poolParams.ProxyAttempts, []*id.ID{target})
	for i := range proxies {
		result, didAbort, err := sendFunc(proxies[i], target)
		if err == nil {
			return result, nil
		} else {
			if didAbort {
				return nil, errors.WithMessagef(err, "Aborted SendToSpecific gateway proxy %s",
					host.GetId().String())
			}
			jww.WARN.Printf("Unable to SendToSpecific proxy %s: %+v", proxies[i].GetId().String(), err)
			err = s.checkReplace(proxies[i].GetId(), err)
			if err != nil {
				jww.ERROR.Printf("Unable to checkReplace: %+v", err)
			}
		}
	}

	return nil, errors.Errorf("Unable to send to specific with proxies")
}

// SendToAny Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (s *Sender) SendToAny(sendFunc func(host *connect.Host) (interface{}, error)) (interface{}, error) {

	proxies := s.getAny(s.poolParams.ProxyAttempts, nil)
	for i := range proxies {
		result, err := sendFunc(proxies[i])
		if err == nil {
			return result, nil
		} else {
			jww.WARN.Printf("Unable to SendToAny %s: %+v", proxies[i].GetId().String(), err)
			err = s.checkReplace(proxies[i].GetId(), err)
			if err != nil {
				jww.ERROR.Printf("Unable to checkReplace: %+v", err)
			}
		}
	}

	return nil, errors.Errorf("Unable to send to any proxies")
}

// SendToPreferred Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (s *Sender) SendToPreferred(targets []*id.ID,
	sendFunc func(host *connect.Host, target *id.ID) (interface{}, error)) (interface{}, error) {

	targetHosts := s.getPreferred(targets)
	for i := range targetHosts {
		result, err := sendFunc(targetHosts[i], targets[i])
		if err == nil {
			return result, nil
		} else {
			jww.WARN.Printf("Unable to SendToPreferred %s: %+v", targetHosts[i].GetId().String(), err)
			err = s.checkReplace(targetHosts[i].GetId(), err)
			if err != nil {
				jww.ERROR.Printf("Unable to checkReplace: %+v", err)
			}
		}
	}

	proxies := s.getAny(s.poolParams.ProxyAttempts, targets)
	for i := range proxies {
		result, err := sendFunc(proxies[i], targets[i%len(targets)])
		if err == nil {
			return result, nil
		} else {
			jww.WARN.Printf("Unable to SendToPreferred proxy %s: %+v", proxies[i].GetId().String(), err)
			err = s.checkReplace(proxies[i].GetId(), err)
			if err != nil {
				jww.ERROR.Printf("Unable to checkReplace: %+v", err)
			}
		}
	}

	return nil, errors.Errorf("Unable to send to any preferred")
}
