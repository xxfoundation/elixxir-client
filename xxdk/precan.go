////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/binary"

	ctidh "git.xx.network/elixxir/ctidh_cgo"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
)

// NewPrecannedCmix creates an insecure user with predetermined keys with
// nodes. It creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information at a
// later date.
func NewPrecannedCmix(precannedID uint, defJSON, storageDir string,
	password []byte) error {
	jww.INFO.Printf("NewPrecannedCmix()")
	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024,
		csprng.NewSystemRNG)
	rngStream := rngStreamGen.GetStream()

	def, err := ParseNDF(defJSON)
	if err != nil {
		return err
	}
	cmixGrp, e2eGrp := DecodeGroups(def)

	userInfo := createPrecannedUser(precannedID, rngStream, e2eGrp)
	store, err := CheckVersionAndSetupStorage(def, storageDir, password,
		userInfo, cmixGrp, e2eGrp, "")
	if err != nil {
		return err
	}

	// Mark the precanned user as finished with permissioning and registered
	// with the network.
	err = store.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return err
	}

	return err
}

const privPQ1PEM = `-----BEGIN CTIDH-1024 PRIVATE KEY-----
AAEAAf4BAQD9AAAA/v3/AwAA/wAA//4A/v8AAP8AAQL//wD9AQAABAD//wAAAgD/
AAECAf//AP4BAP//AAACAQEB/gABAf8A/wAAAP4CBP8AAAEAAAEBAAADAP8AAP7/
/wEAAAAAAAEBAAAA//8AAQABAAABAgAA/wH/AAAAAf8AAA==
-----END CTIDH-1024 PRIVATE KEY-----
`

const privPQ2PEM = `-----BEGIN CTIDH-1024 PRIVATE KEY-----
AP8A/f8A/wD+Af4AAAEAAfsAAAAC/wAA//4A/v8AAgEB/wEA/wIAAv8B/wEAAQL/
/wD//QAAAAAA/wD+AgAAAf//Af3/AAD/AAEAAAIBAQEAAP3/AAAAAAECAf//AAAB
////AAABAAD/AAECAAAA/wABAAEAAQAAAAP/AAACAAAAAA==
-----END CTIDH-1024 PRIVATE KEY-----
`

// MakePrecannedAuthenticatedChannel creates an insecure E2E relationship with a
// precanned user.
func (m *E2e) MakePrecannedAuthenticatedChannel(precannedID uint) (
	contact.Contact, error) {

	rng := m.GetRng().GetStream()
	precanUserInfo := createPrecannedUser(precannedID, rng, m.GetStorage().GetE2EGroup())
	rng.Close()
	precanRecipient, err := buildReceptionIdentity(precanUserInfo.ReceptionID,
		precanUserInfo.ReceptionSalt, precanUserInfo.ReceptionRSA,
		m.GetStorage().GetE2EGroup(), precanUserInfo.E2eDhPrivateKey)
	if err != nil {
		return contact.Contact{}, err
	}
	precanContact := precanRecipient.GetContact()

	myID := binary.BigEndian.Uint64(m.GetReceptionIdentity().ID[:])

	priv1 := ctidh.NewEmptyPrivateKey()
	err = priv1.FromPEMFile(privPQ1PEM)
	if err != nil {
		return contact.Contact{}, err
	}
	priv2 := ctidh.NewEmptyPrivateKey()
	err = priv2.FromPEMFile(privPQ2PEM)
	if err != nil {
		return contact.Contact{}, err
	}

	myPriv := priv1
	theirPriv := priv2
	if myID > uint64(precannedID) {
		myPriv = priv2
		theirPriv = priv1
	}

	theirPub := ctidh.DerivePublicKey(theirPriv)

	// Add the precanned user as a e2e contact
	// FIXME: these params need to be threaded through...
	sesParam := session.GetDefaultParams()
	_, err = m.e2e.AddPartner(precanContact.ID, precanContact.DhPubKey,
		m.e2e.GetHistoricalDHPrivkey(), theirPub,
		myPriv, sesParam, sesParam)

	// Check garbled messages in case any messages arrived before creating
	// the channel
	m.GetCmix().CheckInProgressMessages()

	return precanContact, err
}
