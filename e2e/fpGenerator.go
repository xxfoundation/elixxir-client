package e2e

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
)

// Wrapper which allows the network>manager's fingerprint interface to be
// passed into ratchet without exposing ratchet to buisness logic
// adheres to the CypherHandler interface in session
type fpGenerator struct {
	*manager
}

func (fp *fpGenerator) AddKey(k *session.Cypher) {
	err := fp.net.AddFingerprint(fp.myID, k.Fingerprint(), &processor{
		cy: k,
		m:  fp.manager,
	})
	if err != nil {
		jww.ERROR.Printf("Could not add fingerprint %s: %+v",
			k.Fingerprint(), err)
	}
}

func (fp *fpGenerator) DeleteKey(k *session.Cypher) {
	fp.net.DeleteFingerprint(fp.myID, k.Fingerprint())
}
