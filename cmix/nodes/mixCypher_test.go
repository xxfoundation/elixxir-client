////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"bytes"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
)

// Tests that the encrypted payloads and KMACs generated are consistent.
func Test_mixCypher_Encrypt_Consistency(t *testing.T) {
	const numKeys = 5

	expectedPayload := []byte{220, 95, 160, 88, 229, 136, 42, 254, 239, 32, 57,
		120, 7, 187, 69, 66, 199, 95, 136, 118, 130, 192, 167, 143, 94, 80, 250,
		22, 85, 47, 200, 208, 68, 179, 143, 31, 21, 215, 17, 117, 179, 170, 67,
		59, 14, 158, 116, 249, 10, 116, 166, 127, 168, 26, 11, 41, 129, 166,
		133, 135, 93, 217, 61, 99, 29, 198, 86, 34, 83, 72, 158, 44, 178, 57,
		158, 168, 107, 43, 54, 107, 183, 16, 149, 133, 109, 166, 154, 248, 185,
		218, 32, 11, 200, 191, 240, 197, 27, 21, 82, 198, 42, 109, 79, 28, 116,
		64, 34, 44, 178, 75, 142, 79, 17, 31, 17, 196, 104, 20, 44, 125, 80, 72,
		205, 76, 23, 69, 132, 176, 180, 211, 193, 200, 175, 149, 133, 2, 153,
		114, 21, 239, 107, 46, 237, 41, 48, 188, 241, 97, 89, 65, 213, 218, 73,
		38, 213, 194, 113, 142, 203, 176, 124, 222, 172, 128, 152, 228, 18, 128,
		26, 122, 199, 192, 255, 84, 222, 165, 77, 199, 57, 56, 7, 72, 20, 158,
		133, 90, 63, 68, 145, 54, 34, 223, 152, 157, 105, 217, 30, 111, 83, 4,
		200, 125, 120, 189, 232, 146, 130, 84, 119, 240, 144, 166, 111, 6, 56,
		26, 93, 95, 69, 225, 103, 174, 211, 204, 66, 181, 33, 198, 65, 140, 53,
		255, 37, 120, 204, 59, 128, 70, 54, 228, 26, 197, 107, 186, 22, 93, 189,
		234, 89, 217, 90, 133, 153, 189, 114, 73, 75, 55, 77, 209, 136, 102,
		193, 60, 241, 25, 101, 238, 162, 49, 94, 219, 46, 152, 100, 120, 152,
		131, 78, 128, 226, 47, 21, 253, 171, 40, 122, 161, 69, 56, 102, 63, 89,
		160, 209, 219, 142, 51, 179, 165, 243, 70, 137, 24, 221, 105, 39, 0,
		214, 201, 221, 184, 104, 165, 44, 82, 13, 239, 197, 80, 252, 200, 115,
		146, 200, 51, 63, 173, 88, 163, 3, 214, 135, 89, 118, 99, 197, 98, 80,
		176, 150, 139, 71, 6, 7, 37, 252, 82, 225, 187, 212, 65, 4, 154, 28,
		170, 224, 242, 17, 68, 245, 73, 234, 216, 255, 2, 168, 235, 116, 147,
		252, 217, 85, 157, 38, 243, 43, 213, 250, 219, 124, 86, 155, 129, 99,
		195, 217, 163, 9, 133, 217, 6, 77, 127, 88, 168, 217, 84, 66, 224, 90,
		11, 210, 218, 215, 143, 239, 221, 138, 231, 57, 149, 175, 221, 188, 128,
		169, 28, 215, 39, 147, 36, 52, 146, 75, 20, 228, 230, 197, 1, 80, 38,
		208, 139, 4, 240, 163, 104, 158, 49, 29, 248, 206, 79, 52, 203, 219,
		178, 46, 81, 170, 100, 14, 253, 150, 240, 191, 92, 18, 23, 94, 73, 110,
		212, 237, 84, 86, 102, 32, 78, 209, 207, 213, 117, 141, 148, 218, 209,
		253, 220, 108, 135, 163, 159, 134, 125, 6, 225, 163, 35, 115, 146, 103,
		169, 152, 251, 188, 125, 159, 185, 119, 67, 80, 92, 232, 208, 1, 32,
		144, 250, 32, 187}

	expectedKmacs := [][]byte{
		{110, 235, 79, 128, 16, 94, 181, 95, 101, 152, 187, 204, 87, 236, 211,
			102, 88, 130, 191, 103, 23, 229, 48, 142, 155, 167, 200, 108, 66,
			172, 178, 209},
		{48, 74, 148, 205, 235, 46, 172, 128, 28, 42, 116, 27, 64, 83, 122, 5,
			51, 162, 200, 198, 234, 92, 77, 131, 136, 108, 57, 97, 193, 208,
			148, 217},
		{202, 163, 19, 179, 175, 100, 71, 176, 241, 80, 85, 174, 120, 45, 152,
			117, 82, 193, 203, 188, 158, 60, 111, 217, 64, 47, 219, 169, 100,
			177, 42, 159},
		{66, 121, 20, 21, 206, 142, 3, 75, 229, 94, 197, 4, 117, 223, 245, 117,
			14, 17, 158, 138, 176, 106, 93, 55, 247, 155, 250, 232, 41, 169,
			197, 150},
		{65, 74, 222, 172, 217, 13, 56, 208, 111, 98, 199, 205, 74, 141, 30,
			109, 51, 20, 186, 9, 234, 197, 6, 200, 139, 86, 139, 130, 8, 15, 32,
			209},
	}

	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString(
			"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74"+
				"020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F"+
				"14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6"+
				"F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC200"+
				"7CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD96"+
				"1C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18"+
				"217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF0"+
				"6F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA05101572"+
				"8E5A8AACAA68FFFFFFFFFFFFFFFF", 16),
		large.NewIntFromString("2", 16))

	prng := rand.New(rand.NewSource(42))

	keys := make([]*key, numKeys)

	for i := 0; i < numKeys; i++ {
		keyBytes, _ := csprng.GenerateInGroup(
			cmixGrp.GetPBytes(), cmixGrp.GetP().ByteLen(), prng)
		keys[i] = &key{k: cmixGrp.NewIntFromBytes(keyBytes)}
	}

	salt := make([]byte, 32)
	prng.Read(salt)

	msg := format.NewMessage(cmixGrp.GetP().ByteLen())
	contents := make([]byte, msg.ContentsSize())
	prng.Read(contents)
	msg.SetContents(contents)

	_, pub := ecdh.ECDHNIKE.NewKeypair(csprng.NewSystemRNG())
	rk := mixCypher{keys: keys, g: cmixGrp, ephemeralEdPubKey: pub, ephemeralKeys: make([]bool, len(keys))}

	rid := id.Round(42)

	encMsg, kmacs, _, receivedPub := rk.Encrypt(msg, salt, rid)

	if !bytes.Equal(receivedPub, pub.Bytes()) {
		t.Errorf("Did not receive ephemeral pub key\n\tExpeceted: %+v\n\tReceived: %+v\n", pub.Bytes(), receivedPub)
	}

	if !bytes.Equal(encMsg.Marshal(), expectedPayload) {
		t.Errorf("Encrypted messages do not match.\nexpected: %v\nreceived: %v",
			expectedPayload, encMsg.Marshal())
	}

	if !reflect.DeepEqual(kmacs, expectedKmacs) {
		t.Errorf("KMACs do not match.\nexpected: %v\nreceived: %v",
			expectedKmacs, kmacs)
	}
}
