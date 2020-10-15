package auth

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
)

const saltSize = 32

type format struct {
	data       []byte
	pubkey     []byte
	salt       []byte
	ecrPayload []byte
}

func newFormat(payloadSize, pubkeySize uint, pubkey *cyclic.Int,
	salt []byte) format {

	if len(salt) != saltSize {
		jww.FATAL.Panicf("Salt is wrong size, should be %v, is %v",
			saltSize, len(salt))
	}

	if payloadSize < pubkeySize+saltSize {
		jww.FATAL.Panicf("Size of format is too small, must be big " +
			"enough to contain public key and salt")
	}

	f := buildFormat(make([]byte, payloadSize), pubkeySize)

	copy(f.pubkey, pubkey.LeftpadBytes(uint64(pubkeySize)))
	copy(f.salt, salt)

	return f
}

func buildFormat(data []byte, pubkeySize uint) format {
	f := format{
		data: data,
	}

	f.pubkey = f.data[:pubkeySize]
	f.salt = f.data[pubkeySize : pubkeySize+saltSize]
	f.ecrPayload = f.data[pubkeySize+saltSize:]
	return f
}

func unmarshalFormat(b []byte, pubkeySize uint) (format, error) {
	if uint(len(b)) < pubkeySize+saltSize {
		return format{}, errors.New("Received format too small")
	}

	return buildFormat(b, pubkeySize), nil
}

func (f format) Marshal() []byte {
	return f.data
}

func (f format) GetPubKey(grp *cyclic.Group) *cyclic.Int {
	return grp.NewIntFromBytes(f.pubkey)
}

func (f format) GetSalt() []byte {
	return f.salt
}

func (f format) GetEcrPayload() []byte {
	return f.ecrPayload
}

func (f format) GetEcrPayloadLen() int {
	return len(f.ecrPayload)
}

func (f format) SetEcrPayload(ecr []byte) {
	if len(ecr) != len(f.ecrPayload) {
		jww.FATAL.Panicf("Passed ecr payload incorrect lengh. Expected:"+
			" %v, Recieved: %v", len(f.ecrPayload), len(ecr))
	}

	copy(f.ecrPayload, ecr)
}
