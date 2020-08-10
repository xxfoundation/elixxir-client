package api

import (
	"gitlab.com/xx_network/crypto/signature/rsa"
	"testing"
)

var dummy_key = `-----BEGIN PRIVATE KEY-----
MIIJQQIBADANBgkqhkiG9w0BAQEFAASCCSswggknAgEAAoICAQCrfJyqwVp2Wz6y
FlmPtHBdXffUE1qAkVgZJ1GfErYfO/9wHMYfkihjib9ZFRsOBsIdNEK9Pp/nVJAH
serTLEAwXpvB1EeTXyR/MTh8tVjQuX6RVIiU2gTIKVjZcUs7BaLBLhYXQ8XpjiUT
BltqHa1pj7iU16941i8TxkR0KM90aXfKZBpcJdf0+hc2K1FPVYrtMsuEhOA3cODv
pIU4jRejFSFKJeNwaSw9Y0xf/qTLBz5+ddiSw+3x3AwLuuwTzPVTGdKe57dOVzwq
8N7KTzYprksZgCdEFVeGLy0WahSVYl7IGfzQFHX4J3rrvd5zU/VnL57eqDpB7h57
pyle1roWNToRCOgSn9euSP+Jcq8Pg20i63djDDaL004U41FNTccnpSEIkd6f74h2
LqY64Y1AtMbdozAW5d6EjWAGh0tZXc3aHh8pMBqJ9g/PAFdwCTAn2Q9gmI6NzoIu
F4DFjka0GxrkFyp+f2ijOYmiDwx/ppMnKU4FnTim4FNUT6hKzdob4AvKXhOMEHNe
AoA33h6AvuYcyHaBaOF2pFlNG+3QZNovw40gALse1/ydRHWqGo1g4B9boXXp2Xq3
gcVqEShVMJNJeXKEOHQ8DsFncAr77jCN9qEICSUyICZDl/KVlRSe54i4VJ+6Noa9
S8XmAfzAlsBAdRogFbLHAu45iXipzwIDAQABAoICAEdxL7e3u99JHjKFOySyUIml
R0U0FuUvKBu6lLeHzRXwIffsFOI8OtVVIsGTGGVcjWwrRI6g029FfIeoKKN3cPp1
v8AdlwAfiA3xTI4v4uN6E++p3wjcV1eoWhqkp2ncbDS85XklxAMMNAfcAyOPX5p1
xLlFrhXSbWR4mjYmdl8SPVS1JYI0RecKdbccjtBVW/57xevci6itPxi3WsT3itxn
Riok5L8FIeglQUFQzgjDaNa4c9SZCb1UJjSQ2B9bqOzI+kU3VdeuYiOlm7t/CpqM
wT7LdBBaL894Qflvkkm15LTKltd9XrRWhlBGFrHHTZqCbVZnkXW8JTjwqDyZioZd
Yl1B5VsCZr3TTdJbv4wihtb6gYm7F6uwInWsRqBnfDaEy2Y344SjgTAEGA1JfOOh
DzQN8UEjSPA/yB3K7v3p05PB6zIQt9NmHOZMhesdZNk9I3oLkkB9VZIVE3dtES4F
L7iAJTAtgmgj3LsEI0fY1MLr7ffGR/8voyRHrPxvCT5tnxTcpGVok9wWphV0+Udy
eUxAw5VLsnJotm7syW+zIFHG0VUb7wW3aIYfs9Uc61T1o5kfA2Av9xtHU616pnTh
WgxCmRadB5NDOLAxBwlup6GlvDf1avwcC0MQtuUD55Qu6pomP2gAguxrK/uMiS/f
W0WnRDgtEO+ewv/tuxbZAoIBAQDV34plYw3rgitlGlUHuD1pVC8S2Lds2tZ6HLoc
l4SLVJIQc7jmam37DSjBR/1VW0JpJTMK6VwzbPYSoJpFFHv6sny8NH7y9o3zUq+R
pHNrvAO343EnYcT+TIGn1VlGi9tXyDPGs+ejsuLSnmq0+wtaRnMcaxa+wUCvrOmK
0l6xPuYvHrhAcmlYrGr7DXd+bF3SjLL24tuTmFFlW2A3P3Fg/nfBfbaDjEdPr5dV
vHEsJK9pfMr+tClsZzS4430VFap+WEY58W7Tz7pNJ1/DD0nnyySgPi0I3DVK0p1D
WLdB0gnKvqUNn0Oo73sDKpSfD7Mwdpwyj+zgvflT6kTvFwvTAoIBAQDNQ8EdtIwq
X8RMbz7F88ZHHPX4qbtvMEn7kcgnHsdmIe3/ssltWRfB7S5+B3M/8iY4U84Li4ze
xhjnYK4F9IKsbLUtWCukMjA34W7kaoL+H41NniVTtkIxYBGxmSwMPb9z+WH+5IhD
Ik3GVTTjGXPPNvF+8LgNlZFONdiypw8JwO95PFVHzSqCghMQIQlPqjgKd9xSKg+p
DQrs53tkQEoQMBi92zSrh1HQRVH9mD0KWJYAKSdwEtEhoc/kF8QZ4XIhi8/ByLm/
m/spdoSp9j5Vjy4MRKEYxF1ok9HfwSXsmm2FcJZxeJbiGYruEz4t0kKy0Lmt+8xz
I3jXOXMvYRiVAoIBAE/Cu0FObK2M8RQWeumTG0wBukCEE/wDrQMDXaE2HJc9pe9+
yNEdlgCPishySZcgnqbJ2bxTBTCkjSyrOn1Sw13eXMhvp3yC2LOK/bEKLIVcK+LT
bqqqOqY/8AageVfm5plZL34GL/gLya2UqOTvzu8O4PUTNvtS5QXfLYW5KNlfRMcD
5OEcCg+o1YjlH9BFJ8RS9pc+SXdE0e5D4qEYBveOTykY8g0jLqEYMg8mZOp6j/R+
NtJAbEZiQvZE2KwZVWkjEKWhVZymlqsZaQw80mogh3s/VNo+DZ3m6AFqv4VLiJ1U
9gcbg0cocK7gnWaom0ISqfPtWwEBuE9ESgsEhEMCggEAC29h28jKIjYxllyAL8Dz
49ROM6spAPm8tWIat2s0ipELVDpelFPpSelvtJ+voPlZfbvVd7kvgN2iV4mASF6l
xPtNYJhP3hbZrtNFPT5dy9BwK8nKpI47w8ppUe6JkKkD+G8FMZEDslG/6XOnvZsW
Y43ZCExaxI73iFbhmppJ8S4paSSeT6CzZI/ghf6BKUn/Uz34LS+grbdHS4ldy2j1
d09moXULyx5/xU2HUsxfYisrOBkS1GCH/AqqrTdRumtf01SZn18SUgVbiaTLoThR
oqyWUSKlot6VoZTSlVeKSFMWFN//0ZR5O2FW5wp1ZVIYWyPbpECp1CQ+wCa4LwSG
vQKCAQBcfmjb+R0fVZKXhgig6fjO8LkOFYSYwKnN0ZY53EhPYnGlmD6C9aufihjg
QKdmqP0yJbaKT+DwZYfCmDk3WOTQ5J7rl8yku+I5dX3oY54J3VgQA1/KABrtPmby
Byj2iMMkYutn1ffCsptTd06N4PZ+yU/sQVik3/9R0UVQ3eZqI0Hqon7FED8HWXp5
UJDpahnI/gl8Bl6qtyM17IVh5//VZNMBvZG9cVThlJ3cNfkuuN3CkzWyZM46z/4A
EN240SdmgfmeGSZ4gGmkSTtV/kC7eChAtW/oB/mRJ1QeORSPB+eThnJHla/plYYd
jR+QSAa9eEozCngV6LUagC0YYWDZ
-----END PRIVATE KEY-----`

// Happy path
func TestClient_RegisterForNotifications(t *testing.T) {
	// Initialize client with dummy storage
	storage := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello", "", def)
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
		return
	}

	privKey, err := rsa.LoadPrivateKeyFromPem([]byte(dummy_key))
	if err != nil {
		t.Errorf("Failed to load private key: %+v", err)
	}

	// InitNetwork to gateways and reg server
	err = client.InitNetwork()

	if err != nil {
		t.Errorf("Client failed of connect: %+v", err)
	}

	err = client.GenerateKeys(privKey, "test")
	if err != nil {
		t.Errorf("Failed to properly set up keys: %+v", err)
	}

	token := make([]byte, 32)

	err = client.RegisterForNotifications(token)
	if err != nil {
		t.Errorf("Expected happy path, received error: %+v", err)
	}
}

// Happy path
func TestClient_UnregisterForNotifications(t *testing.T) {
	// Initialize client with dummy storage
	storage := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello", "", def)
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	privKey, err := rsa.LoadPrivateKeyFromPem([]byte(dummy_key))
	if err != nil {
		t.Errorf("Failed to load private key: %+v", err)
		return
	}
	err = client.GenerateKeys(privKey, "test")
	if err != nil {
		t.Errorf("Failed to properly set up keys: %+v", err)
		return
	}
	// InitNetwork to gateways and reg server
	err = client.InitNetwork()

	if err != nil {
		t.Errorf("Client failed of connect: %+v", err)
	}

	err = client.UnregisterForNotifications()
	if err != nil {
		t.Errorf("Expected happy path, received error: %+v", err)
	}
}
