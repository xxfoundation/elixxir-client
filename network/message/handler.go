///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

func (p *handler) handleMessages(stop *stoppable.Single) {
	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case bundle := <-p.messageReception:
			go func() {
				wg := sync.WaitGroup{}
				wg.Add(len(bundle.Messages))
				for i := range bundle.Messages {
					msg := bundle.Messages[i]

					go func() {
						count, ts := p.inProcess.Add(
							msg, bundle.RoundInfo, bundle.Identity)
						wg.Done()
						success := p.handleMessage(msg, bundle)
						if success {
							p.inProcess.Remove(
								msg, bundle.RoundInfo, bundle.Identity)
						} else {
							// Fail the message if any part of the decryption
							// fails, unless it is the last attempts and has
							// been in the buffer long enough, in which case
							// remove it
							if count == p.param.MaxChecksInProcessMessage &&
								netTime.Since(ts) > p.param.InProcessMessageWait {
								p.inProcess.Remove(
									msg, bundle.RoundInfo, bundle.Identity)
							} else {
								p.inProcess.Failed(
									msg, bundle.RoundInfo, bundle.Identity)
							}

						}

					}()
				}
				wg.Wait()
				bundle.Finish()
			}()
		}
	}

}

func (p *handler) handleMessage(ecrMsg format.Message, bundle Bundle) bool {
	fingerprint := ecrMsg.GetKeyFP()
	identity := bundle.Identity
	round := bundle.RoundInfo

	// If we have a fingerprint, process it
	if proc, exists := p.pop(identity.Source, fingerprint); exists {
		proc.Process(ecrMsg, identity, round)
		return true
	}

	services, exists := p.get(
		identity.Source, ecrMsg.GetSIH(), ecrMsg.GetContents())
	if exists {
		for _, t := range services {
			go t.Process(ecrMsg, identity, round)
		}
		if len(services) == 0 {
			jww.WARN.Printf("empty service list for %s: %s",
				ecrMsg.Digest(), ecrMsg.GetSIH())
		}
		return true
	} else {
		// TODO: Delete this else block because it should not be needed.
		jww.INFO.Printf("checking backup %v", identity.Source)
		// //if it does not exist, check against the default fingerprint for the identity
		// forMe = fingerprint2.CheckIdentityFP(ecrMsg.GetSIH(),
		// 	ecrMsgContents, preimage.MakeDefault(identity.Source))
	}

	im := fmt.Sprintf("Message cannot be identify: keyFP: %v, round: %d "+
		"msgDigest: %s, not determined to be for client",
		ecrMsg.GetKeyFP(), bundle.Round, ecrMsg.Digest())
	jww.TRACE.Printf(im)

	p.events.Report(1, "MessageReception", "Garbled", im)

	return false
}
