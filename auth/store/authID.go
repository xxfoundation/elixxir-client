package store

import (
	"encoding/base64"
	"gitlab.com/xx_network/primitives/id"
)

type authIdentity [2 * id.ArrIDLen]byte

func makeAuthIdentity(partner, me *id.ID) authIdentity {
	ph := authIdentity{}
	copy(ph[:id.ArrIDLen], me[:])
	copy(ph[id.ArrIDLen:], partner[:])
	return ph
}

func (ai authIdentity) GetMe() *id.ID {
	me := &id.ID{}
	copy(me[:], ai[:id.ArrIDLen])
	return me
}

func (ai authIdentity) GetPartner() *id.ID {
	partner := &id.ID{}
	copy(partner[:], ai[id.ArrIDLen:])
	return partner
}

func (ai authIdentity) String() string {
	return base64.StdEncoding.EncodeToString(ai[:])
}

func makeReceiveRequestPrefix(aid authIdentity) string {
	return "receiveRequest:" + base64.StdEncoding.EncodeToString(aid[:])
}

func makeSentRequestKey(aid authIdentity) string {
	return "sentRequest:" + base64.StdEncoding.EncodeToString(aid[:])
}
