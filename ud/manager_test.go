///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"testing"
)

var testCert = `-----BEGIN CERTIFICATE-----
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
EnretBzQkeKeBwoB2u6NTiOmUjk=
-----END CERTIFICATE-----
`

var testContact = `<xxc(2)LF2ccT+sdqh0AIKlFFeDOJdnxzbQQYhGStgxhOXmijIDkAZiB9kZo+Dl3bRSbBi5pXZ82rOu2IQXz9+5sspChvoccZqgC/dXGhlesmiNy/EbKxWtptTF4tcNyQxtnmCXg1p/HwKey4G2XDekTw86lq6Lpmj72jozvRWlQisqvWz/5deiPaeFGKDKC0OrrDFnIib7WnKqdYt4XyTKdmObnmbvdCbliZq0zBl7J40qKy5FypYXGlZjStIm0R1qtD4XHMZMsrMJEGxdM55zJdSzknXbR8MNahUrGMyUOTivXLHzojYLht0gFQifKMVWhrDjUoVQV43KOLPmdBwY/2Kc5KvVloDeuDXYY0i7tD63gNIp9JA3gJQUJymDdwqbS13riT1DMHHkdTzKEyGdHS+v2l7AVSlJBiTKuyM00FBNuXhhIcFR7ONFCf8cRPOPPBx3Q6iHNsvsca3KPNhwOJBgaQvHSkjIMsudiR954QbwG9rbi2vxVobIgWYMl5j6vlBS/9rfbE/uLdTEQZfNsLKDCIVCCI4I1bYZxZrDLPrfXTrN6W0sCLE7a/kRBQAAAgA7+LwJqiv9O1ogLnS4TYkSEg==xxc>`

func TestManager_SetAlternativeUserDiscovery(t *testing.T) {
	m := newTestManager(t)

	altAddr := "0.0.0.0:11420"
	err := m.SetAlternativeUserDiscovery([]byte(testCert), []byte(altAddr), []byte(testContact))
	if err != nil {
		t.Fatalf("Unexpected error in SetAlternativeUserDiscovery: %v", err)
	}
}
