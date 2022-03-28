package message

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/crypto/sih"
)

type Service struct {
	Identifier []byte
	Tag        string
	Source     []byte //optional metadata field, only used on reception

	//private field for lazy evaluation of preimage
	//Nil denotes not yet evaluated
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
	// calculate
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

func (si Service) MarshalJSON() ([]byte, error) {
	return json.Marshal(&si)
}

func (si Service) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &si)
}

func (si Service) String() string {
	p := si.preimage()
	return fmt.Sprintf("Tag: %s, Identifier: %s, source: %s, "+
		"preimage:%s", si.Tag, base64.StdEncoding.EncodeToString(si.Identifier),
		base64.StdEncoding.EncodeToString(si.Source),
		base64.StdEncoding.EncodeToString(p[:]))
}
