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

func (s *scheme) FromBytes(serializedRatchet []byte) (Ratchet, error) {
	d := &RatchetDisk{}
	err := cbor.Unmarshal(serializedRatchet, d)
	if err != nil {
		return nil, err
	}
	usedKeys := NewStateVector(d.Size)
	err = usedKeys.Unmarshal(d.UsedKeys)
	if err != nil {
		return nil, err
	}
	r := &ratchet{
		size:           d.Size,
		sharedSecret:   d.SharedSecret,
		salt:           d.Salt,
		usedKeys:       usedKeys,
		fingerprintMap: make(map[format.Fingerprint]uint32),
	}
	fingerprints := r.DeriveFingerprints()
	for i := uint32(0); i < r.size; i++ {
		r.fingerprintMap[fingerprints[i]] = i
	}
	return r, nil
}

func (s *scheme) New(sharedSecret, salt []byte, size uint32) Ratchet {
	r := &ratchet{
		size:           size,
		sharedSecret:   sharedSecret,
		salt:           salt,
		usedKeys:       NewStateVector(size),
		fingerprintMap: make(map[format.Fingerprint]uint32),
	}
	fingerprints := r.DeriveFingerprints()
	for i := uint32(0); i < r.size; i++ {
		r.fingerprintMap[fingerprints[i]] = i
	}
	return r
}

type ratchet struct {
	size           uint32
	sharedSecret   []byte
	salt           []byte // typically the relationship fingerprint
	usedKeys       *StateVector
	fingerprintMap map[format.Fingerprint]uint32
	fingerprints   []format.Fingerprint // not serialized to disk
}

type RatchetDisk struct {
	Size         uint32
	SharedSecret []byte
	Salt         []byte
	UsedKeys     []byte
}

func (r *ratchet) Encrypt(plaintext []byte) (*EncryptedMessage, error) {
	index, err := r.usedKeys.Next()
	if err != nil {
		return nil, err
	}
	fp := deriveKeyFingerprint(r.sharedSecret, index, r.salt)
	key := deriveKey(r.sharedSecret, index, r.salt)
	residue := NewKeyResidue(key)
	keyArr := [32]byte{}
	copy(keyArr[:], key)
	ecrContents := e2e.Crypt(e2e.Key(keyArr), fp, plaintext)
	mac := elixxirhash.CreateHMAC(ecrContents, key)
	return &EncryptedMessage{
		Ciphertext:  append(mac, ecrContents...),
		Residue:     residue,
		Fingerprint: fp,
	}, nil
}

func (r *ratchet) Decrypt(encryptedMessage *EncryptedMessage) (plaintext []byte, err error) {
	index := r.fingerprintMap[encryptedMessage.Fingerprint]
	key := deriveKey(r.sharedSecret, index, r.salt)

	const macSize = 32
	macWanted := encryptedMessage.Ciphertext[:macSize]
	ciphertext := encryptedMessage.Ciphertext[macSize:]

	keyArr := [32]byte{}
	copy(keyArr[:], key)
	if !elixxirhash.VerifyHMAC(ciphertext, macWanted, key) {
		return nil, errors.New("MAC failure")
	}
	return e2e.Crypt(e2e.Key(keyArr), encryptedMessage.Fingerprint, ciphertext), nil
}

func (r *ratchet) Save() ([]byte, error) {
	userKeysBytes, err := r.usedKeys.Marshal()
	if err != nil {
		return nil, err
	}
	d := RatchetDisk{
		SharedSecret: r.sharedSecret,
		Salt:         r.salt,
		Size:         r.size,
		UsedKeys:     userKeysBytes,
	}
	return cbor.Marshal(d)
}

func (r *ratchet) DeriveFingerprints() []format.Fingerprint {
	if r.fingerprints != nil {
		return r.fingerprints
	}
	r.fingerprints = make([]format.Fingerprint, r.size)
	for i := uint32(0); i < r.size; i++ {
		fp := deriveKeyFingerprint(r.sharedSecret, i, r.salt)
		r.fingerprints[i] = fp
	}
	return r.fingerprints
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
