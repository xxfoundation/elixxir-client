////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

// Deployment environment constants for the download-ndf code path
const (
	mainnet = "mainnet"
	release = "release"
	dev     = "dev"
	testnet = "testnet"
)

// URL constants pointing to the NDF of the associated deployment environment
// requested for the download-ndf code path.
const (
	mainNetUrl = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/mainnet.json"
	releaseUrl = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/release.json"
	devUrl     = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/default.json"
	testNetUrl = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/testnet.json"
)

// Certificates for deployment environments. Used to verify NDF signatures.
const (
	mainNetCert = `-----BEGIN CERTIFICATE-----
MIIFqTCCA5GgAwIBAgIUO0qHXSeKrOMucO+Zz82Mf1Zlq4gwDQYJKoZIhvcNAQEL
BQAwgYAxCzAJBgNVBAYTAktZMRQwEgYDVQQHDAtHZW9yZ2UgVG93bjETMBEGA1UE
CgwKeHggbmV0d29yazEPMA0GA1UECwwGRGV2T3BzMRMwEQYDVQQDDAp4eC5uZXR3
b3JrMSAwHgYJKoZIhvcNAQkBFhFhZG1pbnNAeHgubmV0d29yazAeFw0yMTEwMzAy
MjI5MjZaFw0zMTEwMjgyMjI5MjZaMIGAMQswCQYDVQQGEwJLWTEUMBIGA1UEBwwL
R2VvcmdlIFRvd24xEzARBgNVBAoMCnh4IG5ldHdvcmsxDzANBgNVBAsMBkRldk9w
czETMBEGA1UEAwwKeHgubmV0d29yazEgMB4GCSqGSIb3DQEJARYRYWRtaW5zQHh4
Lm5ldHdvcmswggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQD08ixnPWwz
FtBIEWx2SnFjBsdrSWCp9NcWXRtGWeq3ACz+ixiflj/U9U4b57aULeOAvcoC7bwU
j5w3oYxRmXIV40QSevx1z9mNcW3xbbacQ+yCgPPhhj3/c285gVVOUzURLBTNAi9I
EA59zAb8Vy0E6zfq4HRAhH11Q/10QgDjEXuGXra1k3IlemVsouiJGNAtKojNDE1N
x9HnraSEiXzdnV2GDplEvMHaLd3s9vs4XsiLB3VwKyHv7EH9+LOIra6pr5BWw+kD
2qHKGmQMOQe0a7nCirW/k9axH0WiA0XWuQu3U1WfcMEfdC/xn1vtubrdYjtzpXUy
oUEX5eHfu4OlA/zoH+trocfARDyBmTVbDy0P9imH//a6GUKDui9r3fXwEy5YPMhb
dKaNc7QWLPHMh1n25h559z6PqxxPT6UqFFbZD2gTw1sbbpjyqhLbnYguurkxY3jZ
ztW337hROzQ1/abbg/P59JA95Pmhkl8nqqDEf0buOmvMazq3Lwg92nuZ8gsdMKXB
xaEtTTpxhTPOqzc1/XQgScZnc+092MBDh3C2GMxzylOIdk+yF2Gyb+VWPUe29dSa
azzxsDXzRy8y8jaOjdSUWaLa/MgS5Dg1AfHtD55bdvqYzw3NEXIVarpMlzl+Z+6w
jvuwz8GyoMSVe+YEGgvSDvlfY/z19aqneQIDAQABoxkwFzAVBgNVHREEDjAMggp4
eC5uZXR3b3JrMA0GCSqGSIb3DQEBCwUAA4ICAQCp0JDub2w5vZQvFREyA+utZ/+s
XT05j1iTgIRKMa3nofDGERYJUG7FcTd373I2baS70PGx8FF1QuXhn4DNNZlW/SZt
pa1d0pAerqFrIzwOuWVDponYHQ8ayvsT7awCbwZEZE4RhooqS4LqnvtgFu/g7LuM
zkFN8TER7HAUn3P7BujLvcgtqk2LMDz+AgBRszDp/Bw7+1EJDeG9d7hC/stXgDV/
vpD1YDpxSmW4zjezFJqV6OdMOwo9RWVIktK3RXbFc6I5UJZ5kmzPe/I2oPPCBQvD
G3VqFLQe5ik5rXP7SgAN1fL/7KuQna0s42hkV64Z2ymCX69G1ofpgpEFaQLaxLbj
QOun0r8A3NyKvHRIh4K0dFcc3FSOF60Y6k769HKbOPmSDjSSg0qO9GEONBJ8BxAT
IHcHoTAOQoqGehdzepXQSjHsPqTXv3ZFFwCCgO0toI0Qhqwo89X6R3k+i4Kaktr7
mLiPO8s0nq1PZ1XrybKE9BCHkYH1JkUDA+M0pn4QAEx/BuM0QnGXoi1sImW3pEUG
NP7fjkISrD48P8P/TLS45sx5pB8MNGEsRw0lBKmuOdWDmdfhOltB6JxmbhpstNZp
6LVLK6SEOwE76xnHiisR2KyhTTiroUq73BgPFWkWhoJDPbmL1DHgnbdKwwstG8Qu
UGb8k8vh6tzqYZAOKg==
-----END CERTIFICATE-----`
	releaseCert = `-----BEGIN CERTIFICATE-----
MIIFtjCCA56gAwIBAgIJAJnUcpLbGSQiMA0GCSqGSIb3DQEBCwUAMIGMMQswCQYD
VQQGEwJVUzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCUNsYXJlbW9udDEQMA4GA1UE
CgwHRWxpeHhpcjEUMBIGA1UECwwLRGV2ZWxvcG1lbnQxEzARBgNVBAMMCmVsaXh4
aXIuaW8xHzAdBgkqhkiG9w0BCQEWEGFkbWluQGVsaXh4aXIuaW8wHhcNMjAxMTE3
MTkwMTUyWhcNMjIxMTE3MTkwMTUyWjCBjDELMAkGA1UEBhMCVVMxCzAJBgNVBAgM
AkNBMRIwEAYDVQQHDAlDbGFyZW1vbnQxEDAOBgNVBAoMB0VsaXh4aXIxFDASBgNV
BAsMC0RldmVsb3BtZW50MRMwEQYDVQQDDAplbGl4eGlyLmlvMR8wHQYJKoZIhvcN
AQkBFhBhZG1pbkBlbGl4eGlyLmlvMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIIC
CgKCAgEAvtByOoSS8SeMLvvHIuOGfnx0VgweveJHX93LUyJxr1RlVBXCgC5/QOQN
N3dmKWzu4YwaA2jtwaAMhkgdfyOcw6kuqfvQjxv99XRIRKM4GZQkJiym2cnorNu7
hm2/bxmj5TjpP9+vFzbjkJrpRQ80hsV7I9+NKzIhMK4YTgte/F/q9URESlMZxTbb
MFh3s5iiBfBLRNFFsHVdy8OVH+Jv5901cLn+yowaMDLrBMOWGlRROg82ZeRAranX
9X1s+6BclJ/cBe/LcDxGso5sco6UzrWHzpDTnOTzHoamQHYCXtAZP4XbzcqI6A5i
GFM2akuG9Wv3XZZv/6eJRnKS2GLkvv7dtzv+nalxoBKtyIE8ICIVOrb+pVJvY1Id
HOXkK9MEJJ6sZhddipUaQw6hD4I0dNEt30Ugq9zTgFcEnM2R7qKpIDmxrRbcl280
TQGNYgdidzleNdZbjcTvsMVhcxPXCY+bVX1xICD1oJiZZbZFejBvPEfLYyzSdYp+
awX5OnLVSrQtTJu9yz5q3q5pHhxJnqS/CVGLTvzLfmk7BGwRZZuK87LnSixyYfpd
S23qI45AEUINEE0HDZsI+KBq0oVlDB0Z3AZpWauRDqY3o6JIbIOpqmZc6KntyL7j
YCAhbB1tchS47PpbIxUgMMGoR3MBkJutPqtTWCEE3l5jvv0CknUCAwEAAaMZMBcw
FQYDVR0RBA4wDIIKZWxpeHhpci5pbzANBgkqhkiG9w0BAQsFAAOCAgEACLoxE3nh
3VzXH2lQo1QzjKkG/+1m75T0l9Wn9uxa2W/90qBCfim1CPfWUstGdRLBi8gjDevV
zK5HN+Cpz2E22qByeN9fl6rJC4zd1vIdexEre5h7goWoV+qFPhOACElor1tF5UQ2
GD+NFH+Z0ALG1u8db0hBv8NCbtD4YzcQzzINEbs9gp/Sq3cRzkz1wCufFwJwr7+R
0YqZfPj/v/w9G9wSUys1s3i4xr2u87T/bPF68VRg6r1+kXRSRevXd99wKwap52jY
zOwsDGZF9BHMpFVYR/yZhfzSK3F1DmvwuqOsfwSFIjrUjfRlwS28zyZ8rjBq1suD
EAdvYCLDmBSGssNh8E20PHmk5UROYFGEEhlK5ZKj/f1HOmMiOX461XK6HODYyitq
Six2dPi1ZlBJW83DyFqSWJaUR/CluBYmqrWoBX+chv54bU2Y9j/sA/O98wa7trsk
ctzvAcXjhXm6ESRVVD/iZvkW5MP2mkgbDpW3RP9souK5JzbcpC7i3hEcAqPSPgzL
94kHDpYNY7jcGQC4CjPdfBi+Tf6il/QLFRFgyHm2ze3+qrlPT6SQ4hSSH1iXyf4v
tlqu6u77fbF9yaHtq7dvYxH1WioIUxMqbIC1CNgGC1Y/LhzgLRKPSTBCrbQyTcGc
0b5cTzVKxdP6v6WOAXVOEkXTcBPZ4nEZxY0=
-----END CERTIFICATE-----`
	devCert = `-----BEGIN CERTIFICATE-----
MIIF4DCCA8igAwIBAgIUegUvihtQooWNIzsNqj6lucXn6g8wDQYJKoZIhvcNAQEL
BQAwgYwxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJDQTESMBAGA1UEBwwJQ2xhcmVt
b250MRAwDgYDVQQKDAdFbGl4eGlyMRQwEgYDVQQLDAtEZXZlbG9wbWVudDETMBEG
A1UEAwwKZWxpeHhpci5pbzEfMB0GCSqGSIb3DQEJARYQYWRtaW5AZWxpeHhpci5p
bzAeFw0yMTExMzAxODMwMTdaFw0zMTExMjgxODMwMTdaMIGMMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCUNsYXJlbW9udDEQMA4GA1UECgwHRWxp
eHhpcjEUMBIGA1UECwwLRGV2ZWxvcG1lbnQxEzARBgNVBAMMCmVsaXh4aXIuaW8x
HzAdBgkqhkiG9w0BCQEWEGFkbWluQGVsaXh4aXIuaW8wggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQCckGabzUitkySleveyD9Yrxrpj50FiGkOvwkmgN1jF
9r5StN3otiU5tebderkjD82mVqB781czRA9vPqAggbw1ZdAyQPTvDPTj7rmzkByq
QIkdZBMshV/zX1z8oXoNB9bzZlUFVF4HTY3dEytAJONJRkGGAw4FTa/wCkWsITiT
mKvkP3ciKgz7s8uMyZzZpj9ElBphK9Nbwt83v/IOgTqDmn5qDBnHtoLw4roKJkC8
00GF4ZUhlVSQC3oFWOCu6tvSUVCBCTUzVKYJLmCnoilmiE/8nCOU0VOivtsx88f5
9RSPfePUk8u5CRmgThwOpxb0CAO0gd+sY1YJrn+FaW+dSR8OkM3bFuTq7fz9CEkS
XFfUwbJL+HzT0ZuSA3FupTIExyDmM/5dF8lC0RB3j4FNQF+H+j5Kso86e83xnXPI
e+IKKIYa/LVdW24kYRuBDpoONN5KS/F+F/5PzOzH9Swdt07J9b7z1dzWcLnKGtkN
WVsZ7Ue6cuI2zOEWqF1OEr9FladgORcdVBoF/WlsA63C2c1J0tjXqqcl/27GmqGW
gvhaA8Jkm20qLCEhxQ2JzrBdk/X/lCZdP/7A5TxnLqSBq8xxMuLJlZZbUG8U/BT9
sHF5mXZyiucMjTEU7qHMR2UGNFot8TQ7ZXntIApa2NlB/qX2qI5D13PoXI9Hnyxa
8wIDAQABozgwNjAVBgNVHREEDjAMggplbGl4eGlyLmlvMB0GA1UdDgQWBBQimFud
gCzDVFD3Xz68zOAebDN6YDANBgkqhkiG9w0BAQsFAAOCAgEAccsH9JIyFZdytGxC
/6qjSHPgV23ZGmW7alg+GyEATBIAN187Du4Lj6cLbox5nqLdZgYzizVop32JQAHv
N1QPKjViOOkLaJprSUuRULa5kJ5fe+XfMoyhISI4mtJXXbMwl/PbOaDSdeDjl0ZO
auQggWslyv8ZOkfcbC6goEtAxljNZ01zY1ofSKUj+fBw9Lmomql6GAt7NuubANs4
9mSjXwD27EZf3Aqaaju7gX1APW2O03/q4hDqhrGW14sN0gFt751ddPuPr5COGzCS
c3Xg2HqMpXx//FU4qHrZYzwv8SuGSshlCxGJpWku9LVwci1Kxi4LyZgTm6/xY4kB
5fsZf6C2yAZnkIJ8bEYr0Up4KzG1lNskU69uMv+d7W2+4Ie3Evf3HdYad/WeUskG
tc6LKY6B2NX3RMVkQt0ftsDaWsktnR8VBXVZSBVYVEQu318rKvYRdOwZJn339obI
jyMZC/3D721e5Anj/EqHpc3I9Yn3jRKw1xc8kpNLg/JIAibub8JYyDvT1gO4xjBO
+6EWOBFgDAsf7bSP2xQn1pQFWcA/sY1MnRsWeENmKNrkLXffP+8l1tEcijN+KCSF
ek1mr+qBwSaNV9TA+RXVhvqd3DEKPPJ1WhfxP1K81RdUESvHOV/4kdwnSahDyao0
EnretBzQkeKeBwoB2u6NTiOmUjk=
-----END CERTIFICATE-----`
	testNetCert = ``
)
