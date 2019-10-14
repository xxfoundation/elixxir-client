////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"bytes"
	"crypto"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/comms/gateway"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"
)

const NumNodes = 3
const NumGWs = NumNodes
const GWsStartPort = 7900
const RegPort = 5000
const ValidRegCode = "UAV6IWD6"

var RegHandler = api.MockRegistration{}
var RegComms *registration.RegistrationComms

var GWComms [NumGWs]*gateway.GatewayComms

var def *ndf.NetworkDefinition

//Variables ported to get mock permissioning server running
var nodeId *id.Node
var permComms *registration.RegistrationComms

type mockPermission struct {
}

func (i *mockPermission) RegisterUser(registrationCode, test string) (hash []byte, err error) {
	return nil, nil
}

func (i *mockPermission) RegisterNode(ID []byte, ServerAddr, ServerTlsCert,
	GatewayAddr, GatewayTlsCert, RegistrationCode string) error {
	return nil
}

func (i *mockPermission) GetCurrentClientVersion() (string, error) {
	return "0.0.0", nil
}

func (i *mockPermission) GetUpdatedNDF(clientNdfHash []byte) ([]byte, error) {
	ndfData := buildMockNDF()
	globals.Log.INFO.Printf("retndf serialization in mock updateNDF: %v", ndfData.Serialize())
	ndfJson, _ := json.Marshal(ndfData)
	return ndfJson, nil
}

func buildMockNDF() ndf.NetworkDefinition {

	ExampleJSON := `{"Timestamp":"2019-06-04T20:48:48-07:00","gateways":[{"Address":"0.0.0.0:7900","Tls_certificate":"-----BEGIN CERTIFICATE-----\nMIIDgTCCAmmgAwIBAgIJAKLdZ8UigIAeMA0GCSqGSIb3DQEBBQUAMG8xCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEaMBgGA1UEAwwRZ2F0ZXdheSou\nY21peC5yaXAwHhcNMTkwMzA1MTgzNTU0WhcNMjkwMzAyMTgzNTU0WjBvMQswCQYD\nVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJQ2xhcmVtb250\nMRswGQYDVQQKDBJQcml2YXRlZ3JpdHkgQ29ycC4xGjAYBgNVBAMMEWdhdGV3YXkq\nLmNtaXgucmlwMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA9+AaxwDP\nxHbhLmn4HoZu0oUM48Qufc6T5XEZTrpMrqJAouXk+61Jc0EFH96/sbj7VyvnXPRo\ngIENbk2Y84BkB9SkRMIXya/gh9dOEDSgnvj/yg24l3bdKFqBMKiFg00PYB30fU+A\nbe3OI/le0I+v++RwH2AV0BMq+T6PcAGjCC1Q1ZB0wP9/VqNMWq5lbK9wD46IQiSi\n+SgIQeE7HoiAZXrGO0Y7l9P3+VRoXjRQbqfn3ETNL9ZvQuarwAYC9Ix5MxUrS5ag\nOmfjc8bfkpYDFAXRXmdKNISJmtCebX2kDrpP8Bdasx7Fzsx59cEUHCl2aJOWXc7R\n5m3juOVL1HUxjQIDAQABoyAwHjAcBgNVHREEFTATghFnYXRld2F5Ki5jbWl4LnJp\ncDANBgkqhkiG9w0BAQUFAAOCAQEAMu3xoc2LW2UExAAIYYWEETggLNrlGonxteSu\njuJjOR+ik5SVLn0lEu22+z+FCA7gSk9FkWu+v9qnfOfm2Am+WKYWv3dJ5RypW/hD\nNXkOYxVJNYFxeShnHohNqq4eDKpdqSxEcuErFXJdLbZP1uNs4WIOKnThgzhkpuy7\ntZRosvOF1X5uL1frVJzHN5jASEDAa7hJNmQ24kh+ds/Ge39fGD8pK31CWhnIXeDo\nvKD7wivi/gSOBtcRWWLvU8SizZkS3hgTw0lSOf5geuzvasCEYlqrKFssj6cTzbCB\nxy3ra3WazRTNTW4TmkHlCUC9I3oWTTxw5iQxF/I2kQQnwR7L3w==\n-----END CERTIFICATE-----"},{"Address":"0.0.0.0:7901","Tls_certificate":"-----BEGIN CERTIFICATE-----\nMIIDgTCCAmmgAwIBAgIJAKLdZ8UigIAeMA0GCSqGSIb3DQEBBQUAMG8xCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEaMBgGA1UEAwwRZ2F0ZXdheSou\nY21peC5yaXAwHhcNMTkwMzA1MTgzNTU0WhcNMjkwMzAyMTgzNTU0WjBvMQswCQYD\nVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJQ2xhcmVtb250\nMRswGQYDVQQKDBJQcml2YXRlZ3JpdHkgQ29ycC4xGjAYBgNVBAMMEWdhdGV3YXkq\nLmNtaXgucmlwMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA9+AaxwDP\nxHbhLmn4HoZu0oUM48Qufc6T5XEZTrpMrqJAouXk+61Jc0EFH96/sbj7VyvnXPRo\ngIENbk2Y84BkB9SkRMIXya/gh9dOEDSgnvj/yg24l3bdKFqBMKiFg00PYB30fU+A\nbe3OI/le0I+v++RwH2AV0BMq+T6PcAGjCC1Q1ZB0wP9/VqNMWq5lbK9wD46IQiSi\n+SgIQeE7HoiAZXrGO0Y7l9P3+VRoXjRQbqfn3ETNL9ZvQuarwAYC9Ix5MxUrS5ag\nOmfjc8bfkpYDFAXRXmdKNISJmtCebX2kDrpP8Bdasx7Fzsx59cEUHCl2aJOWXc7R\n5m3juOVL1HUxjQIDAQABoyAwHjAcBgNVHREEFTATghFnYXRld2F5Ki5jbWl4LnJp\ncDANBgkqhkiG9w0BAQUFAAOCAQEAMu3xoc2LW2UExAAIYYWEETggLNrlGonxteSu\njuJjOR+ik5SVLn0lEu22+z+FCA7gSk9FkWu+v9qnfOfm2Am+WKYWv3dJ5RypW/hD\nNXkOYxVJNYFxeShnHohNqq4eDKpdqSxEcuErFXJdLbZP1uNs4WIOKnThgzhkpuy7\ntZRosvOF1X5uL1frVJzHN5jASEDAa7hJNmQ24kh+ds/Ge39fGD8pK31CWhnIXeDo\nvKD7wivi/gSOBtcRWWLvU8SizZkS3hgTw0lSOf5geuzvasCEYlqrKFssj6cTzbCB\nxy3ra3WazRTNTW4TmkHlCUC9I3oWTTxw5iQxF/I2kQQnwR7L3w==\n-----END CERTIFICATE-----"},{"Address":"0.0.0.0:7902","Tls_certificate":"-----BEGIN CERTIFICATE-----\nMIIDgTCCAmmgAwIBAgIJAKLdZ8UigIAeMA0GCSqGSIb3DQEBBQUAMG8xCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEaMBgGA1UEAwwRZ2F0ZXdheSou\nY21peC5yaXAwHhcNMTkwMzA1MTgzNTU0WhcNMjkwMzAyMTgzNTU0WjBvMQswCQYD\nVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJQ2xhcmVtb250\nMRswGQYDVQQKDBJQcml2YXRlZ3JpdHkgQ29ycC4xGjAYBgNVBAMMEWdhdGV3YXkq\nLmNtaXgucmlwMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA9+AaxwDP\nxHbhLmn4HoZu0oUM48Qufc6T5XEZTrpMrqJAouXk+61Jc0EFH96/sbj7VyvnXPRo\ngIENbk2Y84BkB9SkRMIXya/gh9dOEDSgnvj/yg24l3bdKFqBMKiFg00PYB30fU+A\nbe3OI/le0I+v++RwH2AV0BMq+T6PcAGjCC1Q1ZB0wP9/VqNMWq5lbK9wD46IQiSi\n+SgIQeE7HoiAZXrGO0Y7l9P3+VRoXjRQbqfn3ETNL9ZvQuarwAYC9Ix5MxUrS5ag\nOmfjc8bfkpYDFAXRXmdKNISJmtCebX2kDrpP8Bdasx7Fzsx59cEUHCl2aJOWXc7R\n5m3juOVL1HUxjQIDAQABoyAwHjAcBgNVHREEFTATghFnYXRld2F5Ki5jbWl4LnJp\ncDANBgkqhkiG9w0BAQUFAAOCAQEAMu3xoc2LW2UExAAIYYWEETggLNrlGonxteSu\njuJjOR+ik5SVLn0lEu22+z+FCA7gSk9FkWu+v9qnfOfm2Am+WKYWv3dJ5RypW/hD\nNXkOYxVJNYFxeShnHohNqq4eDKpdqSxEcuErFXJdLbZP1uNs4WIOKnThgzhkpuy7\ntZRosvOF1X5uL1frVJzHN5jASEDAa7hJNmQ24kh+ds/Ge39fGD8pK31CWhnIXeDo\nvKD7wivi/gSOBtcRWWLvU8SizZkS3hgTw0lSOf5geuzvasCEYlqrKFssj6cTzbCB\nxy3ra3WazRTNTW4TmkHlCUC9I3oWTTxw5iQxF/I2kQQnwR7L3w==\n-----END CERTIFICATE-----"}],"nodes":[{"Id":[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Dsa_public_key":"-----BEGIN PUBLIC KEY-----\nMIIDNDCCAiwCggEBAJ22+1lRtmu2/h4UDx0s5VAjdBYf1lON8WSCGGQvC1xIyPek\nGq36GHMkuHZ0+hgisA8ez4E2lD18VXVyZOWhpE/+AS6ZNuAMHT6TELAcfReYBdMF\niyqfS7b5cWv+YRfGtbPMTZvjQRBK1KgK1slOAF9LmT4U8JHrUXQ78zBQw43iNVZ+\nGzTD1qXAzqoaDzaCE8PRmEPQtLCdy5/HLTnI3kHxvxTUu0Vjyig3FiHK0zJLai05\nIUW+v6x0iAUjb1yi/pK4cc2PnDbTKStVCcqMqneirfx7/XfdpvcRJadFb+oVPkMy\nVqImHGoG7TaTeX55lfrVqrvPvj7aJ0HjdUBK4lsCIQDywxGTdM52yTVpkLRlN0oX\n8j+e01CJvZafYcbd6ZmMHwKCAQBcf/awb48UP+gohDNJPkdpxNmIrOW+JaDiSAln\nBxbGE9ewzuaTL4+qfETSyyRSPaU/vk9uw1lYktGqWMQyigbEahVmLn6qcDod7Pi7\nstBdvi65VsFCozhmHRBGHA0TVHIIUFfzSUMJ/6c8YR94syrbtXQMNhyfNb6QmX2y\nAU4u9apheC9Sq+uL1kMsTdCXvFQjsoXa+2DcNk6BYfSio1rKOhCxxNIDzHakcKM6\n/cvdkpWYWavYtW4XJSUteOrGbnG6muPx3SSHGZh0OTzU2DIYaABlR2Dh40wJ5NFV\nF5+ewNxEc/mWvc5u7Ryr7YtvEW962c9QXfD5mONKsnUUsP/nAoIBAERwUmUlL9YP\nq6MSn+bUr6qNZPsVYoQAo8nTjZWiuSjJa2XWnh7sftnISWkwkiiRxo7qfq3sAiD5\nB8+tM6kONeICBXukldXJerxoVBspYa+RiPuDWy2pwGRDBpfty3QqJOpu5g2ThYFJ\nD5Xu0yCuX8ZJRj33nliI8dQgKdQQva6p2VuXzyRT8LwXMfRwLuSB6Schc9mF8C\nkWCb4m0ujlEKe1xKoKt2zG9b1o7XyaVhxguSUAuEznifMzsEUfuONJOy+XoQELex\nF0wvLzNzABcyxkM3lx52uG41mKgJiV6Z0ZyuBRvt+V3VL/38tPn9lsTaFi8N6/IH\nRyy0bWP5s44=\n-----END PUBLIC KEY-----\n","Address":"0.0.0.0:5900","Tls_certificate":"-----BEGIN CERTIFICATE-----MIIDbDCCAlSgAwIBAgIJAOUNtZneIYECMA0GCSqGSIb3DQEBBQUAMGgxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDAeFwOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDhDwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfsWYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSEtJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uAm3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9bJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEAAaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEAneUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIfU/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2qvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4cyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1RtgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E56m52PyzMNV+2N21IPppKwA==-----END CERTIFICATE-----"},{"Id":[2,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Dsa_public_key":"-----BEGIN PUBLIC KEY-----\nMIIDNDCCAiwCggEBAJ22+1lRtmu2/h4UDx0s5VAjdBYf1lON8WSCGGQvC1xIyPek\nGq36GHMkuHZ0+hgisA8ez4E2lD18VXVyZOWhpE/+AS6ZNuAMHT6TELAcfReYBdMF\niyqfS7b5cWv+YRfGtbPMTZvjQRBK1KgK1slOAF9LmT4U8JHrUXQ78zBQw43iNVZ+\nGzTD1qXAzqoaDzaCE8PRmEPQtLCdy5/HLTnI3kHxvxTUu0Vjyig3FiHK0zJLai05\nIUW+v6x0iAUjb1yi/pK4cc2PnDbTKStVCcqMqneirfx7/XfdpvcRJadFb+oVPkMy\nVqImHGoG7TaTeX55lfrVqrvPvj7aJ0HjdUBK4lsCIQDywxGTdM52yTVpkLRlN0oX\n8j+e01CJvZafYcbd6ZmMHwKCAQBcf/awb48UP+gohDNJPkdpxNmIrOW+JaDiSAln\nBxbGE9ewzuaTL4+qfETSyyRSPaU/vk9uw1lYktGqWMQyigbEahVmLn6qcDod7Pi7\nstBdvi65VsFCozhmHRBGHA0TVHIIUFfzSUMJ/6c8YR94syrbtXQMNhyfNb6QmX2y\nAU4u9apheC9Sq+uL1kMsTdCXvFQjsoXa+2DcNk6BYfSio1rKOhCxxNIDzHakcKM6\n/cvdkpWYWavYtW4XJSUteOrGbnG6muPx3SSHGZh0OTzU2DIYaABlR2Dh40wJ5NFV\nF5+ewNxEc/mWvc5u7Ryr7YtvEW962c9QXfD5mONKsnUUsP/nAoIBAFbADcqA8KQh\nxzgylW6VS1dYYelO5DjPZVVSjfdcbj1twu4ZHDNZLOexpv4nGY8xS6vesELXcVOR\n/CHXgh/3byBZYm0zkrBi/FsJJ3nP2uZ1+QCRldI2KzqcLOWH/CAYj8koork9k1Dp\nFq7rMSDgw4pktqvFj9Eev8dSZuRnoCfZbt/6vxi1r30AYAjDYOwcysqcVyUa1tPa\nLEh3JksttXUCd5cvfqatWedTs5Vxo7ICW1toGBHABYvSJkwK0YFfi5RLw+Oda1sA\njJ+aLcIxQjrpoRC2alXCdwmZXVb+O6zluQctw6LJjt4J704ueSvR4VNNhr0uLYGW\nk7e+WoQCS98=\n-----END PUBLIC KEY-----\n","Address":"0.0.0.0:5901","Tls_certificate":"-----BEGIN CERTIFICATE-----MIIDbDCCAlSgAwIBAgIJAOUNtZneIYECMA0GCSqGSIb3DQEBBQUAMGgxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDAeFwOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDhDwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfsWYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSEtJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uAm3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9bJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEAAaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEAneUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIfU/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2qvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4cyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1RtgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E56m52PyzMNV+2N21IPppKwA==-----END CERTIFICATE-----"},{"Id":[3,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Dsa_public_key":"-----BEGIN PUBLIC KEY-----\nMIIDNTCCAiwCggEBAJ22+1lRtmu2/h4UDx0s5VAjdBYf1lON8WSCGGQvC1xIyPek\nGq36GHMkuHZ0+hgisA8ez4E2lD18VXVyZOWhpE/+AS6ZNuAMHT6TELAcfReYBdMF\niyqfS7b5cWv+YRfGtbPMTZvjQRBK1KgK1slOAF9LmT4U8JHrUXQ78zBQw43iNVZ+\nGzTD1qXAzqoaDzaCE8PRmEPQtLCdy5/HLTnI3kHxvxTUu0Vjyig3FiHK0zJLai05\nIUW+v6x0iAUjb1yi/pK4cc2PnDbTKStVCcqMqneirfx7/XfdpvcRJadFb+oVPkMy\nVqImHGoG7TaTeX55lfrVqrvPvj7aJ0HjdUBK4lsCIQDywxGTdM52yTVpkLRlN0oX\n8j+e01CJvZafYcbd6ZmMHwKCAQBcf/awb48UP+gohDNJPkdpxNmIrOW+JaDiSAln\nBxbGE9ewzuaTL4+qfETSyyRSPaU/vk9uw1lYktGqWMQyigbEahVmLn6qcDod7Pi7\nstBdvi65VsFCozhmHRBGHA0TVHIIUFfzSUMJ/6c8YR94syrbtXQMNhyfNb6QmX2y\nAU4u9apheC9Sq+uL1kMsTdCXvFQjsoXa+2DcNk6BYfSio1rKOhCxxNIDzHakcKM6\n/cvdkpWYWavYtW4XJSUteOrGbnG6muPx3SSHGZh0OTzU2DIYaABlR2Dh40wJ5NFV\nF5+ewNxEc/mWvc5u7Ryr7YtvEW962c9QXfD5mONKsnUUsP/nAoIBAQCN19tTnkS3\nitBQXXR/h8OKl+rliFBLgO6h6GvZL4yQDZFtBAOmkrs3wLoDroJRGCeqz/IUb+JF\njslEr/mpm2kcmK77hr535dq7HsWz1fFl9YyGTaOH055FLSV9QEPAV9j3zWADdQ1v\nuSQll+QfWi6lIibWV4HNQ2ywRFoOY8OBLCJB90UXLeJpaPanpqiM8hjda2VGRDbi\nIixEE2lCOWITydiz2DmvXrLhVGF49+g5MDwbWO65dmasCe//Ff6Z4bJ6n049xv\nVtac8nX6FO3eBsV5d+rG6HZXSG3brCKRCSKYCTX1IkTSiutYxYqvwaluoCjOakh0\nKkqvQ8IeVZ+B\n-----END PUBLIC KEY-----\n","Address":"0.0.0.0:5902","Tls_certificate":"-----BEGIN CERTIFICATE-----MIIDbDCCAlSgAwIBAgIJAOUNtZneIYECMA0GCSqGSIb3DQEBBQUAMGgxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDAeFwOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDhDwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfsWYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSEtJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uAm3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9bJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEAAaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEAneUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIfU/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2qvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4cyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1RtgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E56m52PyzMNV+2N21IPppKwA==-----END CERTIFICATE-----"}],"registration":{"Dsa_public_key":"-----BEGIN PUBLIC KEY-----\nMIIDNDCCAiwCggEBAJ22+1lRtmu2/h4UDx0s5VAjdBYf1lON8WSCGGQvC1xIyPek\nGq36GHMkuHZ0+hgisA8ez4E2lD18VXVyZOWhpE/+AS6ZNuAMHT6TELAcfReYBdMF\niyqfS7b5cWv+YRfGtbPMTZvjQRBK1KgK1slOAF9LmT4U8JHrUXQ78zBQw43iNVZ+\nGzTD1qXAzqoaDzaCE8PRmEPQtLCdy5/HLTnI3kHxvxTUu0Vjyig3FiHK0zJLai05\nIUW+v6x0iAUjb1yi/pK4cc2PnDbTKStVCcqMqneirfx7/XfdpvcRJadFb+oVPkMy\nVqImHGoG7TaTeX55lfrVqrvPvj7aJ0HjdUBK4lsCIQDywxGTdM52yTVpkLRlN0oX\n8j+e01CJvZafYcbd6ZmMHwKCAQBcf/awb48UP+gohDNJPkdpxNmIrOW+JaDiSAln\nBxbGE9ewzuaTL4+qfETSyyRSPaU/vk9uw1lYktGqWMQyigbEahVmLn6qcDod7Pi7\nstBdvi65VsFCozhmHRBGHA0TVHIIUFfzSUMJ/6c8YR94syrbtXQMNhyfNb6QmX2y\nAU4u9apheC9Sq+uL1kMsTdCXvFQjsoXa+2DcNk6BYfSio1rKOhCxxNIDzHakcKM6\n/cvdkpWYWavYtW4XJSUteOrGbnG6muPx3SSHGZh0OTzU2DIYaABlR2Dh40wJ5NFV\nF5+ewNxEc/mWvc5u7Ryr7YtvEW962c9QXfD5mONKsnUUsP/nAoIBAAlELnrXLG0s\n4yAAn7IsVWwY7swDnbBcsIF2cnef4tjm/nNwrFKp5AxYqgeXCiJM8VkyJrotWG50\nnXQwMCR6BsvYrlVt/RmQvR8BSrir62uSLK7hMKm7dXnFvtyFtjp91UwTRbIjxhUQ\nGYnhAzrkCDWo1m54ysqXEGlrVwvRXrCAXiLKPiTEIS+B4GFH9W26SwBxhFLNYSUk\nZZ7+4qwMf9aTu7kIpXTP3hNIyRCjtuZvo5SnymtbLARwTP943hW8MOj0+Ege+m1P\ntey6rkMUGQ86cgK9/7+7Jb+EwW5UxdQtFPUFeNKdQ6zDPS6qbliecUrsc12tdgeg\nhQyuMbyKUuo=\n-----END PUBLIC KEY-----\n","Address":"0.0.0.0:5000","Tls_certificate":"-----BEGIN CERTIFICATE-----MIIDkDCCAnigAwIBAgIJAJnjosuSsP7gMA0GCSqGSIb3DQEBBQUAMHQxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEfMB0GA1UEAwwWcmVnaXN0cmF0aW9uKi5jbWl4LnJpcDAeFwOTAzMDUyMTQ5NTZaFw0yOTAzMDIyMTQ5NTZaMHQxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEfMB0GA1UEAwwWcmVnaXN0cmF0aW9uKi5jbWl4LnJpcDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAOQKvqjdh35o+MECBhCwopJzPlQNmq2iPbewRNtI02bUNK3kLQUbFlYdzNGZS4GYXGc5O+jdi8Slx82r1kdjz5PPCNFBARIsOP/L8r3DGeW+yeJdgBZjm1s3ylkamt4Ajiq/bNjysS6L/WSOp+sVumDxtBEzO/UTU1O6QRnzUphLaiWENmErGvsH0CZVq38Ia58k/QjCAzpUcYi4j2l1fb07xqFcQD8H6SmUM297UyQosDrp8ukdIo31Koxr4XDnnNNsYStC26tzHMeKuJ2Wl+3YzsSyflfM2YEcKE31sqB9DS36UkJ8J84eLsHNImGg3WodFAviDB67+jXDbB30NkMCAwEAAaMlMCMwIQYDVR0RBBowGIIWcmVnaXN0cmF0aW9uKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEAF9mNzk+g+o626Rllt3f3/1qIyYQrYJ0BjSWCKYEFMCgZ4JibAJjAvIajhVYERtltffM+YKcdE2kTpdzJ0YJuUnRfuv6sVnXlVVugUUnd4IOigmjbCdM32k170CYMm0aiwGxl4FrNa8ei7AIax/s1n+sqWq3HeW5LXjnoVb+s3HeCWIuLfcgrurfye8FnNhy14HFzxVYYefIKmL+DPlcGGGm/PPYt3u4a2+rP3xaihc65dTa0u5tf/XPXtPxTDPFj2JeQDFxo7QRREbPD89CtYnwuP937CrkvCKrL0GkW1FViXKqZY9F5uhxrvLIpzhbNrs/EbtweY35XGLDCCMkg==-----END CERTIFICATE-----"},"udb":{"Id":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,3],"Dsa_public_key":"-----BEGIN PUBLIC KEY-----\nMIIDNDCCAiwCggEBAJ22+1lRtmu2/h4UDx0s5VAjdBYf1lON8WSCGGQvC1xIyPek\nGq36GHMkuHZ0+hgisA8ez4E2lD18VXVyZOWhpE/+AS6ZNuAMHT6TELAcfReYBdMF\niyqfS7b5cWv+YRfGtbPMTZvjQRBK1KgK1slOAF9LmT4U8JHrUXQ78zBQw43iNVZ+\nGzTD1qXAzqoaDzaCE8PRmEPQtLCdy5/HLTnI3kHxvxTUu0Vjyig3FiHK0zJLai05\nIUW+v6x0iAUjb1yi/pK4cc2PnDbTKStVCcqMqneirfx7/XfdpvcRJadFb+oVPkMy\nVqImHGoG7TaTeX55lfrVqrvPvj7aJ0HjdUBK4lsCIQDywxGTdM52yTVpkLRlN0oX\n8j+e01CJvZafYcbd6ZmMHwKCAQBcf/awb48UP+gohDNJPkdpxNmIrOW+JaDiSAln\nBxbGE9ewzuaTL4+qfETSyyRSPaU/vk9uw1lYktGqWMQyigbEahVmLn6qcDod7Pi7\nstBdvi65VsFCozhmHRBGHA0TVHIIUFfzSUMJ/6c8YR94syrbtXQMNhyfNb6QmX2y\nAU4u9apheC9Sq+uL1kMsTdCXvFQjsoXa+2DcNk6BYfSio1rKOhCxxNIDzHakcKM6\n/cvdkpWYWavYtW4XJSUteOrGbnG6muPx3SSHGZh0OTzU2DIYaABlR2Dh40wJ5NFV\nF5+ewNxEc/mWvc5u7Ryr7YtvEW962c9QXfD5mONKsnUUsP/nAoIBACvR2lUslz3D\nB/MUo0rHVIHVkhVJCxNjtgTOYgJ9ckArSXQbYzr/fcigcNGjUO2LbK5NFp9GK43C\nrLxMUnJ9nkyIVPaWvquJFZItjcDK3NiNGyD4XyM0eRj4dYeSxQM48hvFbmtbjlXn\n9SQTnGIlr1XnTI4RVHZSQOL6kFJIaLw6wYrQ4w08Ng+p45brp5ercAHnLiftNUWP\nqROhQkdSEpS9LEwfotUSY1jP2AhQfaIMxaeXsZuTU1IYvdhMFRL3DR0r5Ww2Upf8\ng0Ace0mtnsUQ2OG+7MTh2jYIEWRjvuoe3RCz603ujW6g7BfQ1H7f4YFwc5xOOJ3u\nr4dj49dCCjc=\n-----END PUBLIC KEY-----\n"},"E2e":{"Prime":"E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F278DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873847AEF49F66E43873","Small_prime":"02","Generator":"02"},"CMIX":{"Prime":"9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44FFE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE235567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA153E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B","Small_prime":"F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F","Generator":"5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C46A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"}}`
	retNDF, _, _ := ndf.DecodeNDF(ExampleJSON)
	/*
		var grp ndf.Group
		grp.Prime = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
			"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
		grp.Generator = "2"
		grp.SmallPrime = "2"
		retNDF := ndf.NetworkDefinition{Timestamp: time.Now(), Registration: reg, Nodes: Nodes, CMIX: grp, E2E: grp}*/
	return *retNDF
}

// Setups general testing params and calls test wrapper
func TestMain(m *testing.M) {

	os.Exit(testMainWrapper(m))
}

type MockConStatCallback struct{}

func (*MockConStatCallback) Callback(status int, TimeoutSeconds int) { return }

// Make sure NewClient returns an error when called incorrectly.
func TestNewClientNil(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	_, err := NewClient(nil, "", ndfStr, pubKey,
		&MockConStatCallback{})
	if err == nil {
		t.Errorf("NewClient returned nil on invalid (nil, nil) input!")
	}

	_, err = NewClient(nil, "hello", "", "",
		&MockConStatCallback{})
	if err == nil {
		t.Errorf("NewClient returned nil on invalid (nil, 'hello') input!")
	}
}

func TestNewClient(t *testing.T) {
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}

	ndfStr, pubKey := getNDFJSONStr(def, t)

	client, err := NewClient(&d, "hello", ndfStr, pubKey,
		&MockConStatCallback{})
	if err != nil {
		t.Errorf("NewClient returned error: %v", err)
	} else if client == nil {
		t.Errorf("NewClient returned nil Client object")
	}
	for _, gw := range GWComms {
		gw.Disconnect(api.PermissioningAddrID)
	}
}

func TestRegister(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", ndfStr, pubKey,
		&MockConStatCallback{})

	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	err = client.Connect()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, err := client.Register(true, ValidRegCode,
		"", "", "")
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
	for _, gw := range GWComms {
		gw.Disconnect(api.PermissioningAddrID)
	}
}

func TestLoginLogout(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", ndfStr, pubKey,
		&MockConStatCallback{})
	if err != nil {
		t.Errorf("Error starting client: %+v", err)
	}

	// Connect to gateway
	err = client.Connect()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, err := client.Register(true, ValidRegCode,
		"", "", "")
	loginRes, err2 := client.Login(regRes, "")
	if err2 != nil {
		t.Errorf("Login failed: %s", err2.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}
	err = client.StartMessageReceiver()
	if err != nil {
		t.Errorf("Could not start message reciever: %+v", err)
	}
	time.Sleep(200 * time.Millisecond)
	err3 := client.Logout()
	if err3 != nil {
		t.Errorf("Logoutfailed: %s", err3.Error())
	}
	for _, gw := range GWComms {
		gw.Disconnect(api.PermissioningAddrID)
	}
}

type MockListener bool

func (m *MockListener) Hear(msg Message, isHeardElsewhere bool) {
	*m = true
}

// Proves that a message can be received by a listener added with the bindings
func TestListen(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", ndfStr, pubKey,
		&MockConStatCallback{})

	// Connect to gateway
	err = client.Connect()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, _ := client.Register(true, ValidRegCode,
		"", "", "")
	_, err = client.Login(regRes, "")

	if err != nil {
		t.Errorf("Could not log in: %+v", err)
	}

	listener := MockListener(false)
	client.Listen(id.ZeroID[:], int32(cmixproto.Type_NO_TYPE), &listener)
	client.client.GetSwitchboard().Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			MessageType: 0,
			Body:        []byte("stuff"),
		},
		Sender:   id.ZeroID,
		Receiver: client.client.GetCurrentUser(),
	})
	if !listener {
		t.Error("Message not received")
	}
	for _, gw := range GWComms {
		gw.Disconnect(api.PermissioningAddrID)
	}
}

func TestStopListening(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", ndfStr, pubKey,
		&MockConStatCallback{})

	// Connect to gateway
	err = client.Connect()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, _ := client.Register(true, ValidRegCode,
		"", "", "")

	_, err = client.Login(regRes, "")

	if err != nil {
		t.Errorf("Could not log in: %+v", err)
	}

	listener := MockListener(false)
	handle := client.Listen(id.ZeroID[:], int32(cmixproto.Type_NO_TYPE), &listener)
	client.StopListening(handle)
	client.client.GetSwitchboard().Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			MessageType: 0,
			Body:        []byte("stuff"),
		},
		Sender:   id.ZeroID,
		Receiver: id.ZeroID,
	})
	if listener {
		t.Error("Message was received after we stopped listening for it")
	}
}

type MockWriter struct {
	lastMessage []byte
}

func (mw *MockWriter) Write(msg []byte) (int, error) {
	mw.lastMessage = msg
	return len(msg), nil
}

func TestSetLogOutput(t *testing.T) {
	mw := &MockWriter{}
	SetLogOutput(mw)
	msg := "Test logging message"
	globals.Log.CRITICAL.Print(msg)
	if !bytes.Contains(mw.lastMessage, []byte(msg)) {
		t.Errorf("Mock writer didn't get the logging message")
	}
}

func TestParse(t *testing.T) {
	ms := parse.Message{}
	ms.Body = []byte{0, 1, 2}
	ms.MessageType = int32(cmixproto.Type_NO_TYPE)
	ms.Receiver = id.ZeroID
	ms.Sender = id.ZeroID

	messagePacked := ms.Pack()

	msOut, err := ParseMessage(messagePacked)

	if err != nil {
		t.Errorf("Message failed to parse: %s", err.Error())
	}

	if msOut.GetMessageType() != int32(ms.MessageType) {
		t.Errorf("Types do not match after message parse: %v vs %v", msOut.GetMessageType(), ms.MessageType)
	}

	if !reflect.DeepEqual(ms.Body, msOut.GetPayload()) {
		t.Errorf("Bodies do not match after message parse: %v vs %v", msOut.GetPayload(), ms.Body)
	}

}

func getNDFJSONStr(netDef *ndf.NetworkDefinition, t *testing.T) (string, string) {
	ndfBytes, err := json.Marshal(netDef)

	if err != nil {
		t.Errorf("Could not JSON the NDF: %+v", err)
	}

	// Load TLS private key
	privKey, err := rsa.LoadPrivateKeyFromPem([]byte("-----BEGIN PRIVATE KEY-----\n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\nOaRA5ZbdE7g7vxKRV36jT3wvD7W+\n-----END PRIVATE KEY-----\n"))
	if err != nil || privKey == nil {
		t.Error("Failed to load privKey\n")
	}

	// Sign the NDF
	rsaHash := crypto.SHA256.New()
	rsaHash.Write(ndfBytes)
	signature, _ := rsa.Sign(
		crand.Reader, privKey, crypto.SHA256, rsaHash.Sum(nil), nil)

	// Compose network definition string
	ndfStr := string(ndfBytes) + "\n" + base64.StdEncoding.EncodeToString(signature) + "\n"

	return ndfStr, "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n"
}

// Handles initialization of mock registration server,
// gateways used for registration and gateway used for session
func testMainWrapper(m *testing.M) int {

	def = getNDF()

	// Initialize permissioning server
	pAddr := def.Registration.Address
	pHandler := registration.Handler(&mockPermission{})
	permComms = registration.StartRegistrationServer(pAddr, pHandler, nil, nil)

	// Start mock gateways used by registration and defer their shutdown (may not be needed)
	for i := 0; i < NumGWs; i++ {

		gw := ndf.Gateway{
			Address: fmtAddress(GWsStartPort + i),
		}

		def.Gateways = append(def.Gateways, gw)
		GWComms[i] = gateway.StartGateway(gw.Address,
			gateway.NewImplementation(), nil, nil)
	}

	// Start mock registration server and defer its shutdown
	def.Registration = ndf.Registration{
		Address: fmtAddress(RegPort),
	}
	RegComms = registration.StartRegistrationServer(def.Registration.Address,
		&RegHandler, nil, nil)

	for i := 0; i < NumNodes; i++ {
		nIdBytes := make([]byte, id.NodeIdLen)
		nIdBytes[0] = byte(i)
		n := ndf.Node{
			ID: nIdBytes,
		}
		def.Nodes = append(def.Nodes, n)
	}

	defer testWrapperShutdown()
	return m.Run()
}

func testWrapperShutdown() {
	for _, gw := range GWComms {
		gw.Shutdown()
	}
	RegComms.Shutdown()

}

func fmtAddress(port int) string { return fmt.Sprintf("localhost:%d", port) }

func getNDF() *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		E2E: ndf.Group{
			Prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B" +
				"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE" +
				"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F" +
				"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041" +
				"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45" +
				"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209" +
				"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29" +
				"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E" +
				"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2" +
				"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696" +
				"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E" +
				"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873" +
				"847AEF49F66E43873",
			SmallPrime: "2",
			Generator:  "2",
		},
		CMIX: ndf.Group{
			Prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
				"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
				"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
				"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
				"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
				"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
				"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
				"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B",
			SmallPrime: "F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
				"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
				"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
				"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
				"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
				"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
				"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
				"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
		Registration: ndf.Registration{
			Address:        fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000)),
			TlsCertificate: "",
		},
	}
}
