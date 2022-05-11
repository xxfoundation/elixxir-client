////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"bytes"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Tests that process.Process properly decrypts the payload and passes it to the
// callback.
func Test_processor_Process(t *testing.T) {
	rng := csprng.NewSystemRNG()
	rsaPrivKey, err := rsa.GenerateKey(rng, 64)
	if err != nil {
		t.Errorf("Failed to generate RSA key: %+v", err)
	}
	s := &crypto.Symmetric{
		ReceptionID: id.NewIdFromString("channel", id.User, t),
		Name:        "MyChannel",
		Description: "This is my channel that I channel stuff on.",
		Salt:        cmix.NewSalt(rng, 32),
		RsaPubKey:   rsaPrivKey.GetPublic(),
	}

	cbChan := make(chan []byte)
	cb := func(payload []byte, _ receptionID.EphemeralIdentity, _ rounds.Round) {
		cbChan <- payload
	}

	p := &symmetricProcessor{
		s:  s,
		cb: cb,
	}

	msg := format.NewMessage(4092)
	payload := make([]byte, msg.ContentsSize())
	_, _ = rng.Read(payload)
	encryptedPayload, mac, fp := p.s.Encrypt(payload, rng)
	msg.SetContents(encryptedPayload)
	msg.SetMac(mac)
	msg.SetKeyFP(fp)

	p.Process(msg, receptionID.EphemeralIdentity{}, rounds.Round{})

	select {
	case r := <-cbChan:
		if !bytes.Equal(r, payload) {
			t.Errorf("Did not receive expected payload."+
				"\nexpected: %v\nreceived: %v", payload, r)
		}
	case <-time.After(15 * time.Millisecond):
		t.Error("Timed out waiting for listener channel to be called.")
	}
}
