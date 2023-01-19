////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"testing"

	"math/rand"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
)

func TestMakeDebugTag(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	partner, _ := codename.GenerateIdentity(rng)
	dtag := makeDebugTag(partner.PubKey, []byte("hi"), "baseTag")

	require.Equal(t, "baseTag-pY7752Fc7oa4", dtag)
}

func TestCalcDMPayloadLen(t *testing.T) {
	net := &mockClient{}
	plen := calcDMPayloadLen(net)
	require.Equal(t, 2110, plen)
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
