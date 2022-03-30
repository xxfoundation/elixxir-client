package e2e

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
)

type fpGenerator struct {
	*Manager
}

func (fp *fpGenerator) AddKey(k *session.Cypher) {
	err := fp.net.AddFingerprint(fp.myID, k.Fingerprint(), &processor{
		cy: k,
		m:  fp.Manager,
	})
	if err != nil {
		jww.ERROR.Printf("Could not add fingerprint %s: %+v",
			k.Fingerprint(), err)
	}
}

func (fp *fpGenerator) DeleteKey(k *session.Cypher) {
	fp.net.DeleteFingerprint(fp.myID, k.Fingerprint())
}
