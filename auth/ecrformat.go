package auth

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

const ownershipSize = 32

type ecrFormat struct {
	data      []byte
	ownership []byte
	payload   []byte
}

func newEcrFormat(size uint, ownership []byte) ecrFormat {
	if size < ownershipSize {
		jww.FATAL.Panicf("Size too small to hold")
	}

	if len(ownership) != ownershipSize {
		jww.FATAL.Panicf("ownership proof is the wrong size")
	}

	f := buildEcrFormat(make([]byte, size))

	copy(f.ownership, ownership)

	return f

}

func buildEcrFormat(data []byte) ecrFormat {
	f := ecrFormat{
		data: data,
	}

	f.ownership = f.data[:ownershipSize]
	f.payload = f.data[ownershipSize:]
	return f
}

func unmarshalEcrFormat(b []byte) (ecrFormat, error) {
	if len(b) < ownershipSize {
		return ecrFormat{}, errors.New("Received ecr format too small")
	}

	return buildEcrFormat(b), nil
}

func (f ecrFormat) Marshal() []byte {
	return f.data
}

func (f ecrFormat) GetOwnership() []byte {
	return f.ownership
}

func (f ecrFormat) GetPayload() []byte {
	return f.payload
}

func (f ecrFormat) GetPayloadSize() int {
	return len(f.payload)
}

func (f ecrFormat) SetPayload(p []byte) {
	if len(p) != len(f.payload) {
		jww.FATAL.Panicf("Payload is the wrong length")
	}

	copy(f.payload, p)
}
