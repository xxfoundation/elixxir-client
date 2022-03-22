///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"fmt"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/elixxir/crypto/e2e"
	fingerprint2 "gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
)

func (m *Manager) handleMessages(stop *stoppable.Single) {
	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case bundle := <-m.messageReception:
				go func(){
					wg := sync.WaitGroup{}
					wg.Add(len(bundle.Messages))
					for _, msg := range bundle.Messages {
						go func() {
							m.handleMessage(msg, bundle)
							wg.Done()
						}()
					}
					wg.Wait()
					bundle.Finish()
				}()

			}

		}
	}

}

func (m *Manager) handleMessage(ecrMsg format.Message, bundle Bundle) {
	fingerprint := ecrMsg.GetKeyFP()
	// msgDigest := ecrMsg.Digest()
	identity := bundle.Identity

	round := bundle.RoundInfo
	// newID := *id.ID{} // todo use new id systme from ticket
	// 	{
	// 	ID id.ID
	// 	ephID ephemeral.Id
	// }

	//save to garbled
	m.garbledStore.Add(ecrMsg)

	var receptionID interfaces.Identity

	// If we have a fingerprint, process it.
	if proc, exists := m.pop(fingerprint); exists {
		proc.Process(ecrMsg, receptionID, round)
		m.garbledStore.Remove(ecrMsg)
		return
	}

	triggers, exists := m.get(ecrMsg.GetIdentityFP(), ecrMsg.GetContents())
	if exists {
		for _, t := range triggers {
			go t.Process(ecrMsg, receptionID, round)
		}
		if len(triggers) == 0 {
			jww.ERROR.Printf("empty trigger list for %s",
				ecrMsg.GetIdentityFP()) // get preimage
		}
		m.garbledStore.Remove(ecrMsg)
		return
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
	m.Internal.Events.Report(1, "MessageReception", "Garbled", im)
	//denote as active in garbled
	m.garbledStore.Failed(ecrMsg)
}
