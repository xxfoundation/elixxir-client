////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Contains gateway message sending wrappers

package gateway

import (
	"encoding/base64"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
	"strings"
	"time"
)

// getDnsPrefix returns the DNS prefix for the given GwId.
func getDnsPrefix(gwId []byte) string {
	return base64.URLEncoding.EncodeToString(gwId)
}

const gatewayUrl = ".mainnet.cmix.rip:"

// Sender object is used for sending that wraps the HostPool for providing
// destinations.
type Sender interface {
	SendToAny(sendFunc func(host *connect.Host) (interface{}, error),
		stop *stoppable.Single) (interface{}, error)
	SendToPreferred(targets []*id.ID, sendFunc SendToPreferredFunc,
		stop *stoppable.Single, timeout time.Duration) (interface{}, error)
	UpdateNdf(ndf *ndf.NetworkDefinition)
	SetGatewayFilter(f Filter)
	GetHostParams() connect.HostParams
}

type sender struct {
	*HostPool
}

const RetryableError = "Nonfatal error occurred, please retry"

// NewSender Create a new Sender object wrapping a HostPool object
func NewSender(poolParams PoolParams, rng *fastRNG.StreamGenerator,
	ndf *ndf.NetworkDefinition, getter HostManager, storage storage.Session,
	addGateway chan network.NodeGateway) (Sender, error) {

	hostPool, err := newHostPool(
		poolParams, rng, ndf, getter, storage, addGateway)
	if err != nil {
		return nil, err
	}
	return &sender{hostPool}, nil
}

// SendToAny call given sendFunc to any Host in the HostPool, attempting with up
// to numProxies destinations.
func (s *sender) SendToAny(sendFunc func(*connect.Host) (interface{}, error),
	stop *stoppable.Single) (interface{}, error) {

	proxies := s.getAny(s.poolParams.ProxyAttempts, nil)
	for proxy := range proxies {
		result, err := sendFunc(proxies[proxy])
		if stop != nil && !stop.IsRunning() {
			return nil,
				errors.Errorf(stoppable.ErrMsg, stop.Name(), "SendToAny")
		} else if err == nil {
			return result, nil
		} else {
			// Now we must check whether the Host should be replaced
			replaced, checkReplaceErr := s.checkReplace(
				proxies[proxy].GetId(), err)
			if replaced {
				jww.WARN.Printf("Unable to SendToAny, replaced a proxy %s "+
					"with error %s", proxies[proxy].GetId(), err.Error())
			} else {
				if checkReplaceErr != nil {
					jww.WARN.Printf("Unable to SendToAny via %s: %s. "+
						"Unable to replace host: %+v",
						proxies[proxy].GetId(), err.Error(), checkReplaceErr)
				} else {
					jww.WARN.Printf("Unable to SendToAny via %s: %s. "+
						"Did not replace host.",
						proxies[proxy].GetId(), err.Error())
				}
			}

			// End for non-retryable errors
			if !strings.Contains(err.Error(), RetryableError) {
				return nil,
					errors.WithMessage(err, "Received error with SendToAny")
			}
		}
	}

	return nil, errors.Errorf("Unable to send to any proxies")
}

// SendToPreferredFunc is the send function passed into Sender.SendToPreferred.
type SendToPreferredFunc func(host *connect.Host, target *id.ID,
	timeout time.Duration) (interface{}, error)

// SendToPreferred Call given sendFunc to any Host in the HostPool, attempting
// with up to numProxies destinations. Returns an error if the timeout is
// reached.
func (s *sender) SendToPreferred(targets []*id.ID, sendFunc SendToPreferredFunc,
	stop *stoppable.Single, timeout time.Duration) (interface{}, error) {

	startTime := netTime.Now()

	// get the hosts and shuffle randomly
	targetHosts := s.getPreferred(targets)

	// Attempt to send directly to targets if they are in the HostPool
	for i := range targetHosts {
		// Return an error if the timeout duration is reached
		if netTime.Since(startTime) > timeout {
			return nil, errors.Errorf(
				"sending to target in HostPool timed out after %s", timeout)
		}

		remainingTimeout := timeout - netTime.Since(startTime)
		result, err := sendFunc(targetHosts[i], targets[i], remainingTimeout)
		if stop != nil && !stop.IsRunning() {
			return nil, errors.Errorf(
				stoppable.ErrMsg, stop.Name(), "SendToPreferred")
		} else if err == nil {
			return result, nil
		} else {
			// Now we must check whether the Host should be replaced
			replaced, checkReplaceErr := s.checkReplace(targetHosts[i].GetId(), err)
			if replaced {
				jww.WARN.Printf("Unable to SendToPreferred first pass via %s, "+
					"replaced a proxy %s with error %s",
					targets[i], targetHosts[i].GetId(), err.Error())
			} else {
				if checkReplaceErr != nil {
					jww.WARN.Printf("Unable to SendToPreferred first pass %s "+
						"via %s: %s. Unable to replace host: %+v", targets[i],
						targetHosts[i].GetId(), err.Error(), checkReplaceErr)
				} else {
					jww.WARN.Printf("Unable to SendToPreferred first pass %s "+
						"via %s: %s. Did not replace host.",
						targets[i], targetHosts[i].GetId(), err.Error())
				}
			}

			// End for non-retryable errors
			if !strings.Contains(err.Error(), RetryableError) {
				return nil, errors.WithMessage(
					err, "Received error with SendToPreferred")
			}
		}
	}
	//
	//// Build a list of proxies for every target
	//proxies := make([][]*connect.Host, len(targets))
	//for i := 0; i < len(targets); i++ {
	//	proxies[i] = s.getAny(s.poolParams.ProxyAttempts, targets)
	//}
	//
	//// Build a map of bad proxies
	//badProxies := make(map[string]interface{})
	//
	//// Iterate between each target's list of proxies, using the next target for
	//// each proxy
	//for proxyIdx := uint32(0); proxyIdx < s.poolParams.ProxyAttempts; proxyIdx++ {
	//	for targetIdx := range proxies {
	//		// Return an error if the timeout duration is reached
	//		if netTime.Since(startTime) > timeout {
	//			return nil, errors.Errorf("iterating over target's proxies "+
	//				"timed out after %s", timeout)
	//		}
	//
	//		target := targets[targetIdx]
	//		targetProxies := proxies[targetIdx]
	//		if !(int(proxyIdx) < len(targetProxies)) {
	//			jww.WARN.Printf("Failed to send to proxy %d on target %d (%s) "+
	//				"due to not enough proxies (only %d), skipping attempt",
	//				proxyIdx, targetIdx, target, len(targetProxies))
	//			continue
	//		}
	//		proxy := targetProxies[proxyIdx]
	//
	//		// Skip bad proxies
	//		if _, ok := badProxies[proxy.String()]; ok {
	//			continue
	//		}
	//
	//		remainingTimeout := timeout - netTime.Since(startTime)
	//		result, err := sendFunc(proxy, target, remainingTimeout)
	//		if stop != nil && !stop.IsRunning() {
	//			return nil, errors.Errorf(
	//				stoppable.ErrMsg, stop.Name(), "SendToPreferred")
	//		} else if err == nil {
	//			return result, nil
	//		} else if strings.Contains(err.Error(), RetryableError) {
	//			// Retry of the proxy could not communicate
	//			jww.INFO.Printf("Unable to SendToPreferred second pass %s "+
	//				"via %s: non-fatal error received, retrying: %s",
	//				target, proxy, err)
	//			continue
	//		} else {
	//			// Check whether the Host should be replaced
	//			replaced, checkReplaceErr := s.checkReplace(proxy.GetId(), err)
	//			badProxies[proxy.String()] = nil
	//			if replaced {
	//				jww.WARN.Printf("Unable to SendToPreferred second pass "+
	//					"via %s, replaced a proxy %s with error %s",
	//					target, proxy.GetId(), err.Error())
	//			} else {
	//				if checkReplaceErr != nil {
	//					jww.WARN.Printf("Unable to SendToPreferred second "+
	//						"pass %s via %s: %s. Unable to replace host: %+v",
	//						target, proxy.GetId(), err.Error(), checkReplaceErr)
	//				} else {
	//					jww.WARN.Printf("Unable to SendToPreferred second "+
	//						"pass %s via %s: %s. Did not replace host.",
	//						target, proxy.GetId(), err.Error())
	//				}
	//			}
	//
	//			// End for non-retryable errors
	//			if !strings.Contains(err.Error(), RetryableError) {
	//				return nil, errors.WithMessage(
	//					err, "Received error with SendToPreferred")
	//			}
	//		}
	//	}
	//}

	return nil, errors.Errorf("Unable to send to any preferred")
}
