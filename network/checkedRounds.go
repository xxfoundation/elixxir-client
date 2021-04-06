package network

import (
	"crypto/md5"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/xx_network/primitives/id"
)


type idFingerprint [16]byte

type checkedRounds struct{
	lookup map[idFingerprint]map[id.Round]interface{}
}

func newCheckedRounds()*checkedRounds{
	return &checkedRounds{
		lookup: make(map[idFingerprint]map[id.Round]interface{}),
	}
}


func (cr *checkedRounds)Check(identity reception.IdentityUse, rid id.Round)bool{
	idFp := getIdFingerprint(identity)
	if _, exists := cr.lookup[idFp]; !exists{
		cr.lookup[idFp] = make(map[id.Round]interface{})
		cr.lookup[idFp][rid] = nil
		return true
	}

	if _, exists := cr.lookup[idFp][rid]; !exists{
		cr.lookup[idFp][rid] = nil
		return true
	}
	return false
}

func (cr *checkedRounds)Prune(identity reception.IdentityUse, earliestAllowed id.Round){
	idFp := getIdFingerprint(identity)
	if _, exists := cr.lookup[idFp]; !exists{
		return
	}

	for rid, _ := range cr.lookup[idFp]{
		if rid<earliestAllowed{
			delete(cr.lookup[idFp],rid)
		}
	}
}

func getIdFingerprint(identity reception.IdentityUse)idFingerprint{
	h := md5.New()
	h.Write(identity.EphId[:])
	h.Write(identity.Source[:])

	fp := idFingerprint{}
	copy(fp[:], h.Sum(nil))
	return fp
}