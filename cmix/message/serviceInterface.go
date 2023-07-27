////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/base64"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/sih"
	"gitlab.com/xx_network/primitives/id"
)

type Service struct {
	Identifier []byte
	Tag        string
	Metadata   []byte // Optional metadata field, only used on reception

	// Private field for lazy evaluation of preimage
	// A value of nil denotes not yet evaluated
	lazyPreimage *sih.Preimage
}

func (si Service) Hash(_ *id.ID, contents []byte) ([]byte, error) {
	return sih.Hash(si.preimage(), contents), nil
}

func (si Service) HashFromMessageHash(messageHash []byte) []byte {
	return sih.HashFromMessageHash(si.preimage(), messageHash)
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

type CompressedService struct {
	Identifier []byte
	Tags       []string
	Metadata   []byte // when hashed, included in CompressedSIH, when evaluated, recovered

	// Private field for lazy evaluation of preimage
	// A value of nil denotes not yet evaluated
	lazyPreimage *sih.Preimage
}

func (cs CompressedService) ForMe(pickup *id.ID, contents, hash []byte) (
	tags []string, found bool, metadata []byte) {
	matchedTags, metadata, found, err := sih.EvaluateCompressedSIH(
		pickup, sih.GetMessageHash(contents), cs.Identifier, cs.Tags, hash)
	if err != nil {
		jww.WARN.Printf(
			"Failed to evaluate compressed SID for %s: %+v", pickup, err)
	}

	tags = make([]string, 0, len(tags))
	for tag := range matchedTags {
		tags = append(tags, tag)
	}
	return tags, found, metadata
}

func (cs CompressedService) Hash(pickup *id.ID, contents []byte) ([]byte, error) {
	return sih.MakeCompressedSIH(pickup, sih.GetMessageHash(contents),
		cs.Identifier, cs.Tags, cs.Metadata)
}

func (cs CompressedService) preimage() sih.Preimage {
	if cs.lazyPreimage == nil {
		// All compressed services with the same identifier have the
		// same preimage because the tags selection is what option will
		// be triggered on secondarily, only the identifier is the
		// primary trigger
		p := sih.MakePreimage(cs.Identifier, "compressed")
		cs.lazyPreimage = &p
	}

	return *cs.lazyPreimage
}

func (cs CompressedService) String() string {
	p := cs.preimage()
	return fmt.Sprintf("Tags: %s, Identifier: %s, source: %s, "+
		"preimage:%s", cs.Tags, base64.StdEncoding.EncodeToString(cs.Identifier),
		base64.StdEncoding.EncodeToString(cs.Metadata),
		base64.StdEncoding.EncodeToString(p[:]))
}
