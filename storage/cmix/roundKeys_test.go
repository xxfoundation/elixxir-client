package cmix

import (
	"bytes"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"math/rand"
	"reflect"
	"testing"
)

// tests that the encrypted paylaods and kmacs generated are consistent
func TestRoundKeys_Encrypt_Consistency(t *testing.T) {
	const numKeys = 5

	expectedPayload := []byte{80, 118, 187, 96, 114, 221, 253, 46, 231, 113,
		200, 88, 90, 42, 236, 96, 82, 244, 197, 32, 147, 185, 33, 27, 55, 128,
		63, 247, 24, 218, 177, 8, 153, 34, 177, 57, 2, 153, 44, 134, 66, 57,
		212, 140, 254, 125, 34, 173, 58, 39, 130, 130, 12, 114, 81, 254, 120,
		194, 181, 159, 166, 167, 67, 172, 201, 191, 150, 161, 217, 178, 234, 65,
		31, 240, 120, 69, 195, 196, 80, 206, 119, 14, 233, 193, 9, 108, 212,
		157, 13, 160, 48, 171, 244, 106, 109, 48, 216, 117, 60, 98, 166, 26, 5,
		26, 98, 115, 184, 87, 123, 197, 69, 159, 136, 247, 43, 165, 86, 11, 27,
		7, 73, 189, 199, 68, 75, 34, 123, 245, 65, 169, 192, 46, 250, 47, 192,
		238, 211, 196, 26, 254, 33, 53, 92, 9, 138, 197, 34, 209, 102, 58, 170,
		119, 118, 73, 249, 235, 109, 81, 114, 186, 20, 247, 61, 94, 158, 50, 12,
		217, 207, 216, 175, 83, 34, 244, 48, 159, 9, 101, 149, 92, 21, 4, 135,
		91, 14, 142, 43, 5, 140, 197, 63, 216, 105, 20, 73, 38, 38, 250, 158,
		140, 149, 187, 166, 194, 59, 75, 92, 21, 91, 166, 245, 54, 37, 103, 27,
		168, 214, 252, 121, 175, 125, 190, 163, 178, 138, 1, 114, 247, 205, 105,
		14, 248, 177, 89, 190, 205, 10, 109, 193, 189, 73, 117, 239, 179, 10,
		164, 248, 251, 235, 232, 215, 56, 56, 250, 203, 114, 34, 208, 116, 94,
		204, 165, 70, 109, 26, 155, 11, 210, 64, 8, 37, 34, 84, 30, 106, 41, 98,
		135, 63, 62, 225, 212, 251, 245, 36, 238, 166, 142, 76, 192, 46, 169,
		18, 55, 87, 245, 101, 224, 213, 225, 164, 109, 248, 50, 142, 122, 14,
		76, 52, 179, 118, 95, 58, 86, 73, 12, 169, 85, 1, 19, 125, 190, 244,
		231, 233, 95, 72, 101, 178, 230, 107, 59, 109, 220, 114, 155, 138, 96,
		208, 167, 169, 143, 94, 145, 141, 24, 56, 167, 135, 128, 85, 147, 22,
		67, 199, 154, 127, 174, 220, 210, 220, 5, 237, 28, 225, 234, 187, 83,
		124, 215, 185, 38, 149, 87, 1, 29, 109, 31, 132, 145, 85, 90, 195, 226,
		252, 60, 113, 155, 82, 238, 120, 154, 185, 36, 164, 199, 4, 146, 76, 3,
		243, 19, 215, 192, 133, 159, 34, 27, 37, 138, 246, 45, 170, 99, 169, 46,
		253, 98, 203, 52, 242, 203, 106, 141, 75, 140, 90, 118, 38, 162, 107,
		182, 181, 6, 105, 208, 97, 66, 82, 72, 235, 56, 173, 242, 87, 241, 48,
		29, 191, 72, 89, 200, 163, 192, 252, 187, 181, 54, 144, 53, 173, 137,
		142, 19, 207, 3, 207, 169, 12, 148, 198, 225, 195, 118, 85, 153, 159,
		168, 245, 16, 229, 227, 89, 224, 30, 127, 217, 193, 212, 52, 211, 120,
		73, 204, 82, 82, 253, 238, 96, 186, 243, 26, 246, 157, 241, 120, 47,
		170, 83, 175, 58, 179}

	expectedKmacs := [][]byte{
		{241, 132, 2, 131, 104, 92, 89, 120, 177, 8, 201,
			194, 41, 63, 99, 30, 82, 44, 125, 204, 55, 145, 29, 62, 228, 57,
			55, 208, 221, 195, 73, 50},
		{108, 243, 239, 28, 162, 109, 196, 127, 8, 41, 134, 241, 44, 112, 225,
			90, 138, 107, 6, 41, 123, 210, 194, 241, 176, 240, 35, 70, 196,
			149, 48, 77},
		{102, 155, 236, 6, 96, 155, 93, 100, 25, 38, 132, 2, 109, 216, 56, 157,
			60, 100, 99, 226, 123, 181, 99, 157, 115, 215, 104, 243, 48, 161,
			220, 184},
		{154, 237, 87, 227, 221, 68, 206, 8, 163, 133, 253, 96, 96, 220, 215,
			167, 62, 5, 47, 209, 95, 125, 13, 244, 211, 184, 77, 78, 226, 26,
			24, 239},
		{211, 180, 44, 51, 228, 147, 142, 94, 48, 99, 224, 101, 48, 43, 223, 23,
			231, 0, 11, 229, 126, 247, 202, 97, 149, 163, 107, 68, 120, 251, 158,
			33}}

	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1"+
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD"+
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245"+
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED"+
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D"+
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F"+
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D"+
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B"+
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9"+
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510"+
			"15728E5A8AACAA68FFFFFFFFFFFFFFFF", 16),
		large.NewIntFromString("2", 16))

	prng := rand.New(rand.NewSource(42))

	keys := make([]*key, numKeys)

	for i := 0; i < numKeys; i++ {
		keyBytes, _ := csprng.GenerateInGroup(cmixGrp.GetPBytes(), cmixGrp.GetP().ByteLen(), prng)
		keys[i] = &key{
			k: cmixGrp.NewIntFromBytes(keyBytes),
		}
	}

	salt := make([]byte, 32)
	prng.Read(salt)

	msg := format.NewMessage(cmixGrp.GetP().ByteLen())
	contents := make([]byte, msg.ContentsSize())
	prng.Read(contents)
	msg.SetContents(contents)

	rk := RoundKeys{
		keys: keys,
		g:    cmixGrp,
	}

	encMsg, kmacs := rk.Encrypt(msg, salt)

	if !bytes.Equal(encMsg.GetData(), expectedPayload) {
		t.Errorf("Encrypted messages do not match")
	}

	if !reflect.DeepEqual(kmacs, expectedKmacs) {
		t.Errorf("kmacs do not match")
	}
}
