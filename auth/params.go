package auth

import "gitlab.com/elixxir/client/e2e/parse/partition"

type Param struct {
	ReplayRequests bool

	RequestTag string
	ResetTag   string
}

type PartPacket map[trasferID][]part

func (pp PartPacket) Add(transferid, part) {
	list, exist := PartPacket[transferid]
	if exist {
		PartPacket[transferid] = append(list, part)
	} else {
		PartPacket[transferid][]
		part{part}
	}
}
