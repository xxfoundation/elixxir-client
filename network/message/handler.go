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
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/stoppable"
	fingerprint2 "gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

func (p *pickup) handleMessages(stop *stoppable.Single) {
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
						count, ts := p.inProcess.Add(msg, bundle.RoundInfo, bundle.Identity)
						wg.Done()
						success := p.handleMessage(msg, bundle)
						if success {
							p.inProcess.Remove(msg, bundle.RoundInfo, bundle.Identity)
						} else {
							// fail the message if any part of the decryption fails,
							// unless it is the last attempts and has been in the buffer long
							// enough, in which case remove it
							if count == p.param.MaxChecksInProcessMessage &&
								netTime.Since(ts) > p.param.InProcessMessageWait {
								p.inProcess.Remove(msg, bundle.RoundInfo, bundle.Identity)
							} else {
								p.inProcess.Failed(msg, bundle.RoundInfo, bundle.Identity)
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

func (p *pickup) handleMessage(ecrMsg format.Message, bundle Bundle) bool {
	fingerprint := ecrMsg.GetKeyFP()
	identity := bundle.Identity
	round := bundle.RoundInfo

	var receptionID interfaces.Identity

	// If we have a fingerprint, process it.
	if proc, exists := p.pop(identity.Source, fingerprint); exists {
		proc.Process(ecrMsg, receptionID, round)
		return true
	}

	triggers, exists := p.get(identity.Source, ecrMsg.GetIdentityFP(), ecrMsg.GetContents())
	if exists {
		for _, t := range triggers {
			go t.Process(ecrMsg, receptionID, round)
		}
		if len(triggers) == 0 {
			jww.ERROR.Printf("empty trigger list for %s",
				ecrMsg.GetIdentityFP()) // get preimage
		}
		return true
	} else {
		// TODO: delete this else block because it should not be needed.
		jww.INFO.Printf("checking backup %v", preimage.MakeDefault(identity.Source))
		// //if it doesnt exist, check against the default fingerprint for the identity
		// forMe = fingerprint2.CheckIdentityFP(ecrMsg.GetIdentityFP(),
		// 	ecrMsgContents, preimage.MakeDefault(identity.Source))
	}

	if jww.GetLogThreshold() == jww.LevelTrace {
		expectedFP := fingerprint2.IdentityFP(ecrMsg.GetContents(),
			preimage.MakeDefault(identity.Source))
		jww.TRACE.Printf("Message for %d (%s) failed identity "+
			"check: %v (expected-default) vs %v (received)",
			identity.EphId,
			identity.Source, expectedFP, ecrMsg.GetIdentityFP())
	}
	im := fmt.Sprintf("Garbled/RAW Message: keyFP: %v, round: %d"+
		"msgDigest: %s, not determined to be for client",
		ecrMsg.GetKeyFP(), bundle.Round, ecrMsg.Digest())
	p.events.Report(1, "MessageReception", "Garbled", im)
	return false
}
