////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Contains gateway message sending wrappers

package gateway

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	commNetwork "gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
	"strings"
	"testing"
	"time"
)

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
	StartProcesses() stoppable.Stoppable
}

type sender struct {
	*hostPool
}

const RetryableError = "Nonfatal error occurred, please retry"

// NewSender Create a new Sender object wrapping a HostPool object
func NewSender(poolParams Params, rng *fastRNG.StreamGenerator,
	ndf *ndf.NetworkDefinition, getter HostManager,
	storage storage.Session, addChan chan commNetwork.NodeGateway) (
	Sender, error) {

	hp, err := newHostPool(poolParams, rng, ndf,
		getter, storage, addChan, nil)
	if err != nil {
		return nil, err
	}
	return &sender{hp}, nil
}

// NewSender Create a new Sender object wrapping a HostPool object
func NewTestingSender(poolParams Params, rng *fastRNG.StreamGenerator,
	ndf *ndf.NetworkDefinition, getter HostManager,
	storage storage.Session, addChan chan commNetwork.NodeGateway,
	t *testing.T) (Sender, error) {

	if t == nil {
		jww.FATAL.Panicf("can only be called in testing")
	}

	hp, err := newTestingHostPool(poolParams, rng, ndf,
		getter, storage, addChan, nil, t)
	if err != nil {
		return nil, err
	}

	return &sender{hp}, nil
}

// SendToAny call given sendFunc to any Host in the HostPool, attempting with up
// to numProxies destinations.
func (s *sender) SendToAny(sendFunc func(*connect.Host) (interface{}, error),
	stop *stoppable.Single) (interface{}, error) {

	p := s.getPool()

	if err := p.IsReady(); err != nil {
		return nil, errors.WithMessagef(err, "Failed to SendToAny")
	}

	rng := s.rng.GetStream()

	proxies := p.GetAny(s.params.ProxyAttempts, nil, rng)
	rng.Close()
	for proxy := range proxies {
		proxyHost := proxies[proxy]
		result, err := sendFunc(proxyHost)
		if stop != nil && !stop.IsRunning() {
			return nil,
				errors.Errorf(stoppable.ErrMsg, stop.Name(), "SendToAny")
		} else if err == nil {
			return result, nil
		} else {
			// send a signal to remove from the host pool if it is a not
			// allowed error
			if IsGuilty(err) {
				s.Remove(proxyHost)
			}

			// If the send function denotes the error are recoverable,
			// try another host
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

	p := s.getPool()

	if err := p.IsReady(); err != nil {
		return nil, errors.WithMessagef(err, "Failed to SendToPreferred")
	}

	rng := s.rng.GetStream()
	defer rng.Close()

	// get the hosts and shuffle randomly
	targetHosts := p.GetPreferred(targets, rng)

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
			// send a signal to remove from the host pool if it is a not
			// allowed error
			if IsGuilty(err) {
				s.Remove(targetHosts[i])
			}

			// If the send function denotes the error are recoverable,
			// try another host
			if !strings.Contains(err.Error(), RetryableError) {
				return nil,
					errors.WithMessage(err, "Received error with SendToAny")
			}
		}
	}

	//re-get the pool in case it has had an update
	p = s.getPool()
	// Build a list of proxies for every target
	proxies := make([][]*connect.Host, len(targets))
	for i := 0; i < len(targets); i++ {
		proxies[i] = p.GetAny(s.params.ProxyAttempts, targets, rng)
	}

	// Build a map of bad proxies
	badProxies := make(map[string]interface{})

	// Iterate between each target's list of proxies, using the next target for
	// each proxy
	for proxyIdx := uint32(0); proxyIdx < s.params.ProxyAttempts; proxyIdx++ {
		for targetIdx := range proxies {
			// Return an error if the timeout duration is reached
			if netTime.Since(startTime) > timeout {
				return nil, errors.Errorf("iterating over target's proxies "+
					"timed out after %s", timeout)
			}

			target := targets[targetIdx]
			targetProxies := proxies[targetIdx]
			if !(int(proxyIdx) < len(targetProxies)) {
				jww.WARN.Printf("Failed to send to proxy %d on target %d (%s) "+
					"due to not enough proxies (only %d), skipping attempt",
					proxyIdx, targetIdx, target, len(targetProxies))
				continue
			}
			proxy := targetProxies[proxyIdx]

			// Skip bad proxies
			if _, ok := badProxies[proxy.String()]; ok {
				continue
			}

			remainingTimeout := timeout - netTime.Since(startTime)
			result, err := sendFunc(proxy, target, remainingTimeout)
			if stop != nil && !stop.IsRunning() {
				return nil, errors.Errorf(
					stoppable.ErrMsg, stop.Name(), "SendToPreferred")
			} else if err == nil {
				return result, nil
			} else if strings.Contains(err.Error(), RetryableError) {
				// Retry of the proxy could not communicate
				jww.INFO.Printf("Unable to SendToPreferred second pass %s "+
					"via %s: non-fatal error received, retrying: %s",
					target, proxy, err)
				continue
			} else {
				// send a signal to remove from the host pool if it is a not
				// allowed error
				if IsGuilty(err) {
					s.Remove(proxy)
				}

				// If the send function denotes the error are recoverable,
				// try another host
				if !strings.Contains(err.Error(), RetryableError) {
					return nil,
						errors.WithMessage(err, "Received error with SendToAny")
				}

				// End for non-retryable errors
				if !strings.Contains(err.Error(), RetryableError) {
					return nil, errors.WithMessage(
						err, "Received error with SendToPreferred")
				}
			}
		}
	}

	return nil, errors.Errorf("Unable to send to any preferred")
}
