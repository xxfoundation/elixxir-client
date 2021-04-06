package network

import (
	"container/list"
	"crypto/md5"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/xx_network/primitives/id"
)


type idFingerprint [16]byte

type checkedRounds struct{
	lookup map[idFingerprint]*checklist
}

type checklist struct{
	m map[id.Round]interface{}
	l *list.List
}

func newCheckedRounds()*checkedRounds{
	return &checkedRounds{
		lookup: make(map[idFingerprint]*checklist),
	}
}

func (cr *checkedRounds)Check(identity reception.IdentityUse, rid id.Round)bool{
	idFp := getIdFingerprint(identity)
	cl, exists := cr.lookup[idFp]
	if !exists{
		cl = &checklist{
			m: make(map[id.Round]interface{}),
			l: list.New().Init(),
		}
		cr.lookup[idFp]=cl
	}

	if _, exists := cl.m[rid]; !exists{
		cl.m[rid] = nil
		cl.l.PushBack(rid)
		return true
	}
	return false
}

func (cr *checkedRounds)Prune(identity reception.IdentityUse, earliestAllowed id.Round){
	idFp := getIdFingerprint(identity)
	cl, exists := cr.lookup[idFp]
	if !exists {
		return
	}

	e := cl.l.Front()
	for e!=nil {
		if  e.Value.(id.Round)<earliestAllowed{
			delete(cl.m,e.Value.(id.Round))
			lastE := e
			e = e.Next()
			cl.l.Remove(lastE)
		}else{
			break
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