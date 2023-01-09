////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"testing"
	"time"

	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
)

// Adheres to the E2e interface.
type mockE2e struct {
	partnerIDs          []*id.ID
	historicalDHPubkey  *cyclic.Int
	historicalDHPrivkey *cyclic.Int
}

func newMockE2e(t testing.TB) *mockE2e {
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(0))
	return &mockE2e{
		partnerIDs: []*id.ID{
			id.NewIdFromString("partner1", id.User, t),
			id.NewIdFromString("partner2", id.User, t),
			id.NewIdFromString("partner3", id.User, t),
		},
		historicalDHPubkey:  grp.NewInt(45),
		historicalDHPrivkey: grp.NewInt(46),
	}
}
func (m *mockE2e) GetAllPartnerIDs() []*id.ID          { return m.partnerIDs }
func (m *mockE2e) GetHistoricalDHPubkey() *cyclic.Int  { return m.historicalDHPubkey }
func (m *mockE2e) GetHistoricalDHPrivkey() *cyclic.Int { return m.historicalDHPrivkey }

// Adheres to the Session interface.
type mockSession struct {
	regCode                                     string
	transmissionID                              *id.ID
	transmissionSalt                            []byte
	receptionID                                 *id.ID
	receptionSalt                               []byte
	receptionRSA                                rsa.PrivateKey
	transmissionRSA                             rsa.PrivateKey
	transmissionRegistrationValidationSignature []byte
	receptionRegistrationValidationSignature    []byte
	registrationTimestamp                       time.Time
}

func newMockSession(t testing.TB) *mockSession {
	sch := rsa.GetScheme()
	receptionRSA, _ := sch.UnmarshalPrivateKeyPEM([]byte(privKey))
	transmissionRSA, _ := sch.UnmarshalPrivateKeyPEM([]byte(privKey))

	return &mockSession{
		regCode:          "regCode",
		transmissionID:   id.NewIdFromString("transmission", id.User, t),
		transmissionSalt: []byte("transmissionSalt"),
		receptionID:      id.NewIdFromString("reception", id.User, t),
		receptionSalt:    []byte("receptionSalt"),
		receptionRSA:     receptionRSA,
		transmissionRSA:  transmissionRSA,
		transmissionRegistrationValidationSignature: []byte("transmissionSig"),
		receptionRegistrationValidationSignature:    []byte("receptionSig"),
		registrationTimestamp:                       time.Date(2012, 12, 21, 22, 8, 41, 0, time.UTC),
	}

}
func (m mockSession) GetRegCode() (string, error)        { return m.regCode, nil }
func (m mockSession) GetTransmissionID() *id.ID          { return m.transmissionID }
func (m mockSession) GetTransmissionSalt() []byte        { return m.transmissionSalt }
func (m mockSession) GetReceptionID() *id.ID             { return m.receptionID }
func (m mockSession) GetReceptionSalt() []byte           { return m.receptionSalt }
func (m mockSession) GetReceptionRSA() rsa.PrivateKey    { return m.receptionRSA }
func (m mockSession) GetTransmissionRSA() rsa.PrivateKey { return m.transmissionRSA }
func (m mockSession) GetTransmissionRegistrationValidationSignature() []byte {
	return m.transmissionRegistrationValidationSignature
}
func (m mockSession) GetReceptionRegistrationValidationSignature() []byte {
	return m.receptionRegistrationValidationSignature
}
func (m mockSession) GetRegistrationTimestamp() time.Time { return m.registrationTimestamp }

// Adheres to the UserDiscovery interface.
type mockUserDiscovery struct {
	facts fact.FactList
}

func newMockUserDiscovery() *mockUserDiscovery {
	return &mockUserDiscovery{facts: fact.FactList{
		{"myUserName", fact.Username},
		{"hello@example.com", fact.Email},
		{"6175555212", fact.Phone},
		{"name", fact.Nickname},
	}}
}
func (m mockUserDiscovery) GetFacts() fact.FactList { return m.facts }

const privKey = `-----BEGIN PRIVATE KEY-----
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
OaRA5ZbdE7g7vxKRV36jT3wvD7W+
-----END PRIVATE KEY-----`
