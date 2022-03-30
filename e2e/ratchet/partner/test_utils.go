package partner

import "gitlab.com/elixxir/client/e2e/ratchet/partner/session"

type mockCyHandler struct {
}

func (m mockCyHandler) AddKey(k *session.Cypher) {
	return
}

func (m mockCyHandler) DeleteKey(k *session.Cypher) {
	return
}
