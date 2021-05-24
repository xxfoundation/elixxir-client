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
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

// Sender Object used for sending that wraps the HostPool for providing destinations
type Sender struct {
	*HostPool
}

// NewSender Create a new Sender object wrapping a HostPool object
func NewSender(poolParams PoolParams, rng *fastRNG.StreamGenerator, ndf *ndf.NetworkDefinition, getter HostManager,
	storage *storage.Session, addGateway chan network.NodeGateway) (*Sender, error) {

	hostPool, err := newHostPool(poolParams, rng, ndf, getter, storage, addGateway)
	if err != nil {
		return nil, err
	}
	return &Sender{hostPool}, nil
}

// SendToAny Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (s *Sender) SendToAny(sendFunc func(host *connect.Host) (interface{}, error)) (interface{}, error) {

	proxies := s.getAny(s.poolParams.ProxyAttempts, nil)
	for i := range proxies {
		result, err := sendFunc(proxies[i])
		if err == nil {
			return result, nil
		} else {
			jww.WARN.Printf("Unable to SendToAny %s: %s", proxies[i].GetId().String(), err)
			_, err = s.checkReplace(proxies[i].GetId(), err)
			if err != nil {
				jww.ERROR.Printf("Unable to checkReplace: %+v", err)
			}
		}
	}

	return nil, errors.Errorf("Unable to send to any proxies")
}

// SendToPreferred Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (s *Sender) SendToPreferred(targets []*id.ID,
	sendFunc func(host *connect.Host, target *id.ID) (interface{}, bool, error)) (interface{}, error) {

	// Get the hosts and shuffle randomly
	targetHosts := s.getPreferred(targets)

	// Attempt to send directly to targets if they are in the HostPool
	for i := range targetHosts {
		result, didAbort, err := sendFunc(targetHosts[i], targets[i])
		if err == nil {
			return result, nil
		} else {
			if didAbort {
				return nil, errors.WithMessagef(err, "Aborted SendToPreferred gateway %s",
					targetHosts[i].GetId().String())
			}
			jww.WARN.Printf("Unable to SendToPreferred %s via %s: %s",
				targets[i], targetHosts[i].GetId(), err)
			_, err = s.checkReplace(targetHosts[i].GetId(), err)
			if err != nil {
				jww.ERROR.Printf("Unable to checkReplace: %+v", err)
			}
		}
	}

	// Build a list of proxies for every target
	proxies := make([][]*connect.Host, len(targets))
	for i := 0; i < len(targets); i++ {
		proxies[i] = s.getAny(s.poolParams.ProxyAttempts, targets)
	}

	// Build a map of bad proxies
	badProxies := make(map[string]interface{})

	// Iterate between each target's list of proxies, using the next target for each proxy
	for proxyIdx := uint32(0); proxyIdx < s.poolParams.ProxyAttempts; proxyIdx++ {
		for targetIdx := range proxies {
			target := targets[targetIdx]
			targetProxies := proxies[targetIdx]
			proxy := targetProxies[proxyIdx]

			// Skip bad proxies
			if _, ok := badProxies[proxy.String()]; ok {
				continue
			}

			result, didAbort, err := sendFunc(targetProxies[proxyIdx], target)
			if err == nil {
				return result, nil
			} else {
				if didAbort {
					return nil, errors.WithMessagef(err, "Aborted SendToPreferred gateway proxy %s",
						proxy.GetId().String())
				}
				jww.WARN.Printf("Unable to SendToPreferred %s via proxy "+
					"%s: %s", target, proxy.GetId(), err)
				wasReplaced, err := s.checkReplace(proxy.GetId(), err)
				if err != nil {
					jww.ERROR.Printf("Unable to checkReplace: %+v", err)
				}
				// If the proxy was replaced, add as a bad proxy
				if wasReplaced {
					badProxies[proxy.String()] = nil
				}
			}
		}
	}

	return nil, errors.Errorf("Unable to send to any preferred")
}
