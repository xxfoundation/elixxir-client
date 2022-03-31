package partner

import (
	"encoding/base64"
	"gitlab.com/xx_network/primitives/id"
)

type ManagerIdentity [2 * id.ArrIDLen]byte

func MakeManagerIdentity(partner, me *id.ID) ManagerIdentity {
	ph := ManagerIdentity{}
	copy(ph[:id.ArrIDLen], me[:])
	copy(ph[id.ArrIDLen:], partner[:])
	return ph
}

func (mi ManagerIdentity) GetMe() *id.ID {
	me := &id.ID{}
	copy(me[:], mi[:id.ArrIDLen])
	return me
}

func (mi ManagerIdentity) GetPartner() *id.ID {
	partner := &id.ID{}
	copy(partner[:], mi[id.ArrIDLen:])
	return partner
}

func (mi ManagerIdentity) String() string {
	return base64.StdEncoding.EncodeToString(mi[:])
}
