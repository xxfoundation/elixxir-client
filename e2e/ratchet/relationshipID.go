package ratchet

import (
	"gitlab.com/xx_network/primitives/id"
)

type relationshipIdentity [2 * id.ArrIDLen]byte

func makeRelationshipIdentity(partner, me *id.ID) relationshipIdentity {
	ph := relationshipIdentity{}
	copy(ph[:id.ArrIDLen], me[:])
	copy(ph[id.ArrIDLen:], partner[:])
	return ph
}

func (ri relationshipIdentity) GetMe() *id.ID {
	me := &id.ID{}
	copy(me[:], ri[:id.ArrIDLen])
	return me
}

func (ri relationshipIdentity) GetPartner() *id.ID {
	partner := &id.ID{}
	copy(partner[:], ri[id.ArrIDLen:])
	return partner
}
