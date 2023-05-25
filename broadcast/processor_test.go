////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"bytes"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"testing"
	"time"
)

// Tests that process.Process properly decrypts the payload and passes it to the
// callback.
func Test_processor_Process(t *testing.T) {
	rng := csprng.NewSystemRNG()
	name := "MyChannel"
	desc := "This is my channel that I channel stuff on."
	msg := format.NewMessage(4096 / 8)
	s, _, err := crypto.NewChannel(name, desc, crypto.Public, msg.ContentsSize(), rng)
	if err != nil {
		t.Fatalf("could not generate channel: %+v", err)
	}

	cbChan := make(chan []byte, 1)
	cb := func(payload, _ []byte, _ []string, _ [2]byte, _ receptionID.EphemeralIdentity, _ rounds.Round) {
		cbChan <- payload
	}

	p := &processor{
		c:      s,
		cb:     cb,
		method: Symmetric,
	}

	payload := make([]byte, s.GetMaxSymmetricPayloadSize(msg.ContentsSize()))
	_, _ = rng.Read(payload)
	encryptedPayload, mac, fp, err := p.c.EncryptSymmetric(payload, msg.ContentsSize(), rng)
	if err != nil {
		t.Fatalf("could not encrypt data: %+v", err)
	}
	msg.SetContents(encryptedPayload)
	msg.SetMac(mac)
	msg.SetKeyFP(fp)

	p.Process(msg, nil, nil, receptionID.EphemeralIdentity{}, rounds.Round{})

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
