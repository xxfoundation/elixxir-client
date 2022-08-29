///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"testing"
)

var testCert = `-----BEGIN CERTIFICATE-----
MIIDbDCCAlSgAwIBAgIJAOUNtZneIYECMA0GCSqGSIb3DQEBBQUAMGgxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx
GzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJp
cDAeFw0xOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVT
MRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNV
BAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDh
Dwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfs
WYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSE
tJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uA
m3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9
bJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEA
AaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEA
neUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIf
U/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2
qvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4
cyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1R
tgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E5
6m52PyzMNV+2N21IPppKwA==
-----END CERTIFICATE-----
`

var testKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA8/RbJWR9kD8IR3YOAqly52fSgOEPCyObHwsENaxSBDPIusWJ
WKB9SZR5+SdQfd1MmOaIm0aouxrIKdZ2pYsc8li8V+xZhbQgYX3dQb/g3lQBY7ii
QLcJiApKLh6Sl/DbOBmrSCZvNOddwcX2B+ZDuMlQpIS0k9I3Ner49l4lML3MkBvg
uPxDMfXysMzwNovbLWezRjYe7R50EWf86pq8Ekpv+4CbcASOp46ELsRBSNVqZaVM
E+3jT4H/poUlfkFibYkWRGwYEjIFUvjoy8LTYyDNNP1skgl/CqYFpOu7B6Y2DMa7
smTTeGqpLVNQS9Ijr/HxtWhtreitwqR/0ewOMQIDAQABAoIBAQDxobnZ2qQoCNbZ
eUw9RLtEC2jMMJ8m6FiQMeg0hX8jHGuY22nD+ArAo6kAqPkoAdcJp2XtbtpXoRpb
nkochCLixBOhfr/ZF+Xuyq0pn7VKYaiSrmE/ekydi5uX/L40cuOfuIUXzMHfg78w
3DRp9KBlWjlfCvaVZ+U5qYh49h0eHSF0Le9Q6+gAZ/FCGfLYHI+hpmMK+8QlXvD4
XVnjTEH8dUGUarVGxIw6p0ZsF7T6kYgPHG5e2zc6roIqfOWBG4MBkkYUdPECqY1O
sHbZl5TVUK8GdYhX+U3dnCmC4L96n4djVEGQx68fQB6rJ2WE2VPlpDpj+M4wenPX
MpB02ftBAoGBAP08KOnGrNz40nZ6yfVSRBfkyJHsXAV5XTHtw2u7YSPSXN9tq+7Z
AIVRaO9km2SxWfNDdkEge0xmQjKe/1CkjiwqBBjsDI6P1yYQIReoXndSAZ4JmS8P
6IzdDpv4vDjmC55Y+c5+uFQn8+1zdeYHwQYie+5LxsAKQxo1wbaNaN6JAoGBAPae
QWhZbiEVkftznrPpfAW4Fl/ZlCAIonvn6uYXM/1TMDGqPIOZyfYB6BIFjkdT9aHP
ZhZtFgWNnAyta37EM+FGnDtBTmFJ3tl4gqZWLIK6T2csinrDsdv/s7VpduB0yAE0
sfWuRZoBfEpUof37TS//YR6Ibm/G0IS8LnrSMIhpAoGAYKluFI45vb9c1szX+kSE
qXoy9UB7f7trz3sqdRz5X2sU+FQspOdAQ6NnormMd0sbQrglk4aKigcejaQTYPzv
J/yBw+GWiXRuc6EEgLtME8/Bvkl7p3MzGVHoGbFAZ5eoJ7Fe6WuFgNofSiwgfMXI
8EaJd9SE8Rj5tC+A2eXwecECgYAxXv05Jq4lcWwIKt1apyNtAa15AtXkk9XzeDpO
VdbSoBTF3I7Aycjktvz+np4dKXHDMwH8+1mtQuw6nX0no5+/OaONOUW3tFIotzdw
lU/T2/iJbyFJ8mNo54fSiYqC5N4lX6dAx+KnMiTvvIGxlt2c/kMzGZ0CQ4r7B7FG
ZU3SAQKBgQCxE34846J4kH6jRsboyZVkdDdzXQ+NeICJXcaHM2okjnT50IG6Qpwd
0yPXN6xvYW5L+FVb80NfD1y8LkmBerNEMpcwwDL1ZhgiKWQmITESphnYpm3GV9pe
1vIMaHV6GeX+q/RcLu2kU4hJbH6HDRJxtdkmw/gdSo9vphDgB6qALw==
-----END RSA PRIVATE KEY-----
`

var testContact = `<xxc(2)LF2ccT+sdqh0AIKlFFeDOJdnxzbQQYhGStgxhOXmijIDkAZiB9kZo+Dl3bRSbBi5pXZ82rOu2IQXz9+5sspChvoccZqgC/dXGhlesmiNy/EbKxWtptTF4tcNyQxtnmCXg1p/HwKey4G2XDekTw86lq6Lpmj72jozvRWlQisqvWz/5deiPaeFGKDKC0OrrDFnIib7WnKqdYt4XyTKdmObnmbvdCbliZq0zBl7J40qKy5FypYXGlZjStIm0R1qtD4XHMZMsrMJEGxdM55zJdSzknXbR8MNahUrGMyUOTivXLHzojYLht0gFQifKMVWhrDjUoVQV43KOLPmdBwY/2Kc5KvVloDeuDXYY0i7tD63gNIp9JA3gJQUJymDdwqbS13riT1DMHHkdTzKEyGdHS+v2l7AVSlJBiTKuyM00FBNuXhhIcFR7ONFCf8cRPOPPBx3Q6iHNsvsca3KPNhwOJBgaQvHSkjIMsudiR954QbwG9rbi2vxVobIgWYMl5j6vlBS/9rfbE/uLdTEQZfNsLKDCIVCCI4I1bYZxZrDLPrfXTrN6W0sCLE7a/kRBQAAAgA7+LwJqiv9O1ogLnS4TYkSEg==xxc>`

func TestManager_SetAlternativeUserDiscovery(t *testing.T) {
	isReg := uint32(1)

	// Create a new Private Key to use for signing the Fact
	rng := csprng.NewSystemRNG()
	cpk, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		t.Fatal(err)
	}

	comms, err := client.NewClientComms(nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}

	// Create our Manager object
	m := Manager{
		comms:      comms,
		net:        newTestNetworkManager(t),
		privKey:    cpk,
		registered: &isReg,
	}

	altAddr := "0.0.0.0:11420"
	err = m.SetAlternativeUserDiscovery([]byte(testCert), []byte(altAddr), []byte(testContact))
	if err != nil {
		t.Fatalf("Unexpected error in SetAlternativeUserDiscovery: %v", err)
	}
}
