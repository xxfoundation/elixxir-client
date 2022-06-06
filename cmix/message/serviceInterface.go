package message

import (
	"encoding/base64"
	"fmt"
	"gitlab.com/elixxir/crypto/sih"
)

type Service struct {
	Identifier []byte
	Tag        string
	Metadata   []byte // Optional metadata field, only used on reception

	// Private field for lazy evaluation of preimage
	// A value of nil denotes not yet evaluated
	lazyPreimage *sih.Preimage
}

func (si Service) Hash(contents []byte) []byte {
	preimage := si.preimage()
	return sih.Hash(preimage, contents)
}

func (si Service) HashFromMessageHash(messageHash []byte) []byte {
	preimage := si.preimage()
	return sih.HashFromMessageHash(preimage, messageHash)
}

func (si Service) preimage() sih.Preimage {
	if si.lazyPreimage == nil {
		p := sih.MakePreimage(si.Identifier, si.Tag)
		si.lazyPreimage = &p
	}

	return *si.lazyPreimage
}

func (si Service) ForMe(contents, hash []byte) bool {
	return sih.ForMe(si.preimage(), contents, hash)
}

func (si Service) ForMeFromMessageHash(messageHash, hash []byte) bool {
	return sih.ForMeFromMessageHash(si.preimage(), messageHash, hash)
}

func (si Service) String() string {
	p := si.preimage()
	return fmt.Sprintf("Tag: %s, Identifier: %s, source: %s, "+
		"preimage:%s", si.Tag, base64.StdEncoding.EncodeToString(si.Identifier),
		base64.StdEncoding.EncodeToString(si.Metadata),
		base64.StdEncoding.EncodeToString(p[:]))
}
