///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// cmix.go cMix functions for the auth module

package auth

import (
	"fmt"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// getMixPayloadSize calculates the payload size of a cMix Message based on the
// total message size.
// TODO: Maybe move this to primitives and export it?
// FIXME: This can only vary per cMix network target, and it could be scoped
//        to a Client instance.
func getMixPayloadSize(primeSize int) int {
	return 2*primeSize - format.AssociatedDataSize - 1
}

// sendAuthRequest is a helper to send the cMix Message after the request
// is created.
func sendAuthRequest(recipient *id.ID, contents, mac []byte, primeSize int,
	fingerprint format.Fingerprint, net interfaces.NetworkManager,
	cMixParams params.CMIX, reset bool) (id.Round, error) {
	cmixMsg := format.NewMessage(primeSize)
	cmixMsg.SetKeyFP(fingerprint)
	cmixMsg.SetMac(mac)
	cmixMsg.SetContents(contents)

	jww.INFO.Printf("Requesting Auth with %s, msgDigest: %s",
		recipient, cmixMsg.Digest())
	if reset {
		cMixParams.IdentityPreimage = preimage.GenerateRequest(recipient)
	} else {
		cMixParams.IdentityPreimage = preimage.GenerateReset(recipient)
	}

	cMixParams.DebugTag = "auth.Request"
	round, _, err := net.SendCMIX(cmixMsg, recipient, cMixParams)
	if err != nil {
		// if the send fails just set it to failed, it will
		// but automatically retried
		return 0, errors.WithMessagef(err, "Auth Request with %s "+
			"(msgDigest: %s) failed to transmit: %+v", recipient,
			cmixMsg.Digest(), err)
	}

	em := fmt.Sprintf("Auth Request with %s (msgDigest: %s) sent"+
		" on round %d", recipient, cmixMsg.Digest(), round)
	jww.INFO.Print(em)
	net.GetEventManager().Report(1, "Auth", "RequestSent", em)
	return round, nil
}
