////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
)

func TestMakeDebugTag(t *testing.T) {
}

func TestCalcDMPayloadLen(t *testing.T) {
}

func TestCreateCMIXFields(t *testing.T) {
	rng := csprng.NewSystemRNG()
	payloadSize := 100
	msg := []byte("HelloWorld")
	txt := make([]byte, payloadSize)
	for i := 0; i < payloadSize; i += len(msg) {
		copy(txt[i:], msg)
	}

	fpBytes, encryptedPayload, mac, err := createCMIXFields(txt,
		payloadSize, rng)

	require.NoError(t, err)

	packet := format.NewMessage(100)
	packet.SetContents(encryptedPayload)
	packet.SetMac(mac)
	fp := format.Fingerprint{}
	copy(fp[:], fpBytes)
	packet.SetKeyFP(fp)

	finalContents := reconstructCiphertext(packet)
	require.Equal(t, txt, finalContents[:len(txt)])
}
