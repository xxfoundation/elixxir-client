package ratchet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash"

	"github.com/fxamacker/cbor/v2"
	"golang.org/x/crypto/blake2b"

	elixxirhash "gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"

	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/e2e"
)

const (
	residueSalt      = `e2eKeyResidueSalt`
	KeyResidueLength = 32
)

func NewScheme() *scheme {
	return &scheme{}
}

type scheme struct {
}

func (s *scheme) New(myPrivateKey nike.PrivateKey, theirPublicKey nike.PublicKey) Ratchet {
	return &ratchet{}
}

type ratchet struct {
	sharedSecret []byte

	index uint32

	fingerprintMapSize uint
	fingerprintMap     map[format.Fingerprint]int

	usedKeys *utility.StateVector

	salt []byte // typically the relationship fingerprint
}

type RatchetDisk struct {
	SharedSecret       []byte
	Index              uint32
	FingerprintMapSize uint
	FingerprintMap     map[format.Fingerprint]int
	UsedKeys           []byte
	Salt               []byte
}

func New(myPrivateKey nike.PrivateKey, theirPublicKey nike.PublicKey, fingerprintMapSize uint) Ratchet {
	r := &ratchet{
		usedKeys:           utility.NewStateVector(kv, blah, fubar),
		sharedSecret:       myPrivateKey.DeriveSecret(theirPublicKey),
		fingerprintMapSize: fingerprintMapSize,
		fingerprintMap:     make(map[format.Fingerprint]int),
	}
	return r
}

func (r *ratchet) Encrypt(plaintext []byte) *EncryptedMessage {
	fp := deriveKeyFingerprint(r.sharedSecret, r.index, r.salt)
	key := deriveKey(r.sharedSecret, r.index, r.salt)
	r.index++
	residue := NewKeyResidue(key)
	keyArr := [32]byte{}
	copy(keyArr[:], key)
	ecrContents := e2e.Crypt(e2e.Key(keyArr), fp, plaintext)
	mac := elixxirhash.CreateHMAC(ecrContents, key)
	return &EncryptedMessage{
		Ciphertext:  append(mac, ecrContents...),
		Residue:     residue,
		Fingerprint: fp,
	}
}

func (r *ratchet) Decrypt(encryptedMessage *EncryptedMessage) (plaintext []byte, err error) {
	const macSize = 32
	macWanted := encryptedMessage.Ciphertext[:macSize]
	ciphertext := encryptedMessage.Ciphertext[macSize:]
	fp := deriveKeyFingerprint(r.sharedSecret, r.index, r.salt)
	key := deriveKey(r.sharedSecret, r.index, r.salt)
	keyArr := [32]byte{}
	copy(keyArr[:], key)
	if !elixxirhash.VerifyHMAC(ciphertext, macWanted, key) {
		return nil, errors.New("MAC failure")
	}
	defer r.usedKeys.Use(r.index)
	r.index++
	return e2e.Crypt(e2e.Key(keyArr), fp, ciphertext), nil
}

func (r *ratchet) Save() ([]byte, error) {
	usedKeys, err := r.usedKeys.Marshal()
	if err != nil {
		return nil, err
	}
	d := RatchetDisk{
		SharedSecret:       r.sharedSecret,
		Index:              r.index,
		FingerprintMapSize: r.fingerprintMapSize,
		FingerprintMap:     r.fingerprintMap,
		UsedKeys:           usedKeys,
		Salt:               r.salt,
	}
	return cbor.Marshal(d)
}

// derive creates a bit key from a key id and a byte slice by hashing them and
// all the passed salts with the passed hash function. it will have the size
// of the output of the hash function
func derive(h hash.Hash, data []byte, id uint32, salts ...[]byte) []byte {
	keyIdBytes := make([]byte, binary.MaxVarintLen32)
	n := binary.PutUvarint(keyIdBytes, uint64(id))
	h.Write(data)
	h.Write(keyIdBytes[:n])
	for _, salt := range salts {
		h.Write(salt)
	}
	return h.Sum(nil)
}

// deriveKey derives a single key at position keynum using blake2B on the concatenation
// of the first half of the cyclic basekey and the keynum and the salts
// Key = H(First half of base key | keyNum | salt[0] | salt[1] | ...)
func deriveKey(basekey []byte, keyNum uint32, salts ...[]byte) []byte {
	data := basekey
	data = data[:len(data)/2]

	h, err := blake2b.New256(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create hash for "+
			"DeriveKey: %s", err))
	}

	return derive(h, data, keyNum, salts...)
}

// deriveKeyFingerprint derives a single key fingerprint at position keynum using blake2B on
// the concatenation of the second half of the cyclic basekey and the keynum
// and the salts
// Fingerprint = H(Second half of base key | userID | keyNum | salt[0] | salt[1] | ...)
func deriveKeyFingerprint(dhkey []byte, keyNum uint32, salts ...[]byte) format.Fingerprint {
	data := dhkey
	data = data[len(data)/2:]
	h, err := blake2b.New256(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create hash for "+
			"DeriveKeyFingerprint(): %s", err))
	}
	fpBytes := derive(h, data, keyNum, salts...)
	fp := format.Fingerprint{}
	copy(fp[:], fpBytes)
	fp[0] &= 0x7f // accomodate existing cMix API design
	return fp
}

// NewKeyResidue returns a residue of a Key. The
// residue is the hash of the key with the residueSalt.
func NewKeyResidue(key []byte) []byte {
	h := elixxirhash.DefaultHash()
	h.Write(key[:])
	h.Write([]byte(residueSalt))
	kr := h.Sum(nil)
	return kr[:]
}
