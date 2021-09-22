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
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"strings"
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
func (s *Sender) SendToAny(sendFunc func(host *connect.Host) (interface{}, error), stop *stoppable.Single) (interface{}, error) {

	proxies := s.getAny(s.poolParams.ProxyAttempts, nil)
	for proxy := range proxies {
		result, err := sendFunc(proxies[proxy])
		if stop != nil && !stop.IsRunning() {
			return nil, errors.Errorf(stoppable.ErrMsg, stop.Name(), "SendToAny")
		}else if err==nil{
			return result, nil
		} else if strings.Contains(err.Error(),"unable to connect to target host") {
			// Retry of the proxy could not communicate
			jww.WARN.Printf("Unable to SendToAny via %s: proxy could not contact requested host: %s",
				proxies[proxy].GetId().String(), err)
		}else if replaced, checkReplaceErr := s.checkReplace(proxies[proxy].GetId(), err); replaced{
			if checkReplaceErr!=nil{
				jww.WARN.Printf("Unable to SendToAny, replaced a proxy %s with error %s",
					proxies[proxy].GetId().String(), checkReplaceErr)
			}else{
				jww.WARN.Printf("Unable to SendToAny, replaced a proxy %s",
					proxies[proxy].GetId().String())
			}
		}else{
			return nil, errors.WithMessage(err,"Received error from remote")
		}
	}

	return nil, errors.Errorf("Unable to send to any proxies")
}

// SendToPreferred Call given sendFunc to any Host in the HostPool, attempting with up to numProxies destinations
func (s *Sender) SendToPreferred(targets []*id.ID,
	sendFunc func(host *connect.Host, target *id.ID) (interface{}, error),
	stop *stoppable.Single) (interface{}, error) {

	// Get the hosts and shuffle randomly
	targetHosts := s.getPreferred(targets)

	// Attempt to send directly to targets if they are in the HostPool
	for i := range targetHosts {
		result, err := sendFunc(targetHosts[i], targets[i])
		if stop != nil && !stop.IsRunning() {
			return nil, errors.Errorf(stoppable.ErrMsg, stop.Name(), "SendToPreferred")
		} else if err == nil {
			return result, nil
		} else if strings.Contains(err.Error(),"unable to connect to target host") {
			// Retry of the proxy could not communicate
			jww.WARN.Printf("Unable to SendToPreferred first pass %s via %s: %s, " +
				"proxy could not contact requested host",
				targets[i], targetHosts[i].GetId(), err)
			continue
		} else if replaced, checkReplaceErr := s.checkReplace(targetHosts[i].GetId(), err); replaced {
			if checkReplaceErr!=nil{
				jww.WARN.Printf("Unable to SendToPreferred first pass %s via %s, " +
					"proxy failed, was replaced with error: %s",
					targets[i], targetHosts[i].GetId(), checkReplaceErr)
			}else{
				jww.WARN.Printf("Unable to SendToPreferred first pass %s via %s, " +
					"proxy failed, was replaced",
					targets[i], targetHosts[i].GetId())
			}
			jww.WARN.Printf("Unable to SendToPreferred first pass %s via %s: %s, proxy failed, was replaced",
				targets[i], targetHosts[i].GetId(), checkReplaceErr)
			continue
		} else {
			jww.WARN.Printf("Unable to SendToPreferred first pass %s via %s: %s, comm returned an error",
				targets[i], targetHosts[i].GetId(), err)
			return result, err
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
			if !(int(proxyIdx) < len(targetProxies)) {
				jww.WARN.Printf("Failed to send to proxy %d on target %d (%s) "+
					"due to not enough proxies (only %d), skipping attempt", proxyIdx,
					targetIdx, target, len(targetProxies))
				continue
			}
			proxy := targetProxies[proxyIdx]

			// Skip bad proxies
			if _, ok := badProxies[proxy.String()]; ok {
				continue
			}

			result, err := sendFunc(proxy, target)
			if stop != nil && !stop.IsRunning() {
				return nil, errors.Errorf(stoppable.ErrMsg, stop.Name(), "SendToPreferred")
			} else if err == nil {
				return result, nil
			} else if strings.Contains(err.Error(),"unable to connect to target host") {
				// Retry of the proxy could not communicate
				jww.WARN.Printf("Unable to SendToPreferred second pass %s via %s: %s," +
					" proxy could not contact requested host",
					target, proxy, err)
				continue
			} else if replaced, checkReplaceErr := s.checkReplace(proxy.GetId(), err); replaced {
				if checkReplaceErr!=nil{
					jww.WARN.Printf("Unable to SendToPreferred second pass %s via %s," +
						"proxy failed, was replaced with error: %s", target, proxy.GetId(),
						checkReplaceErr)
				}else{
					jww.WARN.Printf("Unable to SendToPreferred second pass %s via %s, " +
						"proxy failed, was replaced", target, proxy.GetId())
				}

				badProxies[proxy.String()] = nil
				continue
			} else {
				jww.WARN.Printf("Unable to SendToPreferred second pass %s via %s: %s, comm returned an error",
					target, proxy.GetId(), err)
				return result, err
			}
		}
	}

	return nil, errors.Errorf("Unable to send to any preferred")
}