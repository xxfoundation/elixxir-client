package storage

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/xx_network/primitives/ndf"
)

const baseNdfKey = "baseNdf"

func (s *Session) SetBaseNDF(def *ndf.NetworkDefinition) {
	err := utility.SaveNDF(s.kv, baseNdfKey, def)
	if err != nil {
		jww.FATAL.Printf("Failed to dave the base NDF: %s", err)
	}
	s.baseNdf = def
}

func (s *Session) GetBaseNDF() *ndf.NetworkDefinition {
	if s.baseNdf != nil {
		return s.baseNdf
	}
	def, err := utility.LoadNDF(s.kv, baseNdfKey)
	if err != nil {
		jww.FATAL.Printf("Could not load the base NDF: %s", err)
	}

	s.baseNdf = def
	return def
}
