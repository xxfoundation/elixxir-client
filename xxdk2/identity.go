////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk2

import (
	"encoding/json"
	"gitlab.com/elixxir/primitives/fact"

	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const idVersion = 0

// ReceptionIdentity is used by the E2e object for managing identities used for
// message pickup.
type ReceptionIdentity struct {
	ID            *id.ID
	RSAPrivatePem []byte
	Salt          []byte
	DHKeyPrivate  []byte
	E2eGrp        []byte
}

// StoreReceptionIdentity stores the given identity in Cmix storage with the
// given key. This is the ideal way to securely store identities, as the caller
// of this function is only required to store the given key separately rather
// than the keying material.
func StoreReceptionIdentity(key string, identity ReceptionIdentity, net *Cmix) error {
	marshalledIdentity, err := identity.Marshal()
	if err != nil {
		return err
	}

	return net.GetStorage().Set(key, &versioned.Object{
		Version:   idVersion,
		Timestamp: netTime.Now(),
		Data:      marshalledIdentity,
	})
}

// LoadReceptionIdentity loads the given identity in Cmix storage with the given
// key.
func LoadReceptionIdentity(key string, net *Cmix) (ReceptionIdentity, error) {
	storageObj, err := net.GetStorage().Get(key)
	if err != nil {
		return ReceptionIdentity{}, err
	}

	return UnmarshalReceptionIdentity(storageObj.Data)
}

// Marshal returns the JSON representation of a ReceptionIdentity.
func (r ReceptionIdentity) Marshal() ([]byte, error) {
	return json.Marshal(&r)
}

// UnmarshalReceptionIdentity takes in a marshalled ReceptionIdentity and
// converts it to an object.
func UnmarshalReceptionIdentity(marshaled []byte) (ReceptionIdentity, error) {
	newIdentity := ReceptionIdentity{}
	return newIdentity, json.Unmarshal(marshaled, &newIdentity)
}

// GetDHKeyPrivate returns the DHKeyPrivate.
func (r ReceptionIdentity) GetDHKeyPrivate() (*cyclic.Int, error) {
	dhKeyPriv := &cyclic.Int{}
	err := dhKeyPriv.UnmarshalJSON(r.DHKeyPrivate)
	return dhKeyPriv, err
}

// GetRSAPrivateKey returns the RSAPrivatePem.
func (r ReceptionIdentity) GetRSAPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.LoadPrivateKeyFromPem(r.RSAPrivatePem)
}

// GetGroup returns the cyclic.Group.
func (r ReceptionIdentity) GetGroup() (*cyclic.Group, error) {
	grp := &cyclic.Group{}
	return grp, grp.UnmarshalJSON(r.E2eGrp)
}

// MakeReceptionIdentity generates a new cryptographic identity for receiving
// messages.
func MakeReceptionIdentity(net *Cmix) (ReceptionIdentity, error) {
	rng := net.GetRng().GetStream()
	defer rng.Close()
	grp := net.GetStorage().GetE2EGroup()

	// Make RSA Key
	rsaKey, err := rsa.GenerateKey(rng, rsa.DefaultRSABitLen)
	if err != nil {
		return ReceptionIdentity{}, err
	}

	// Make salt
	salt := make([]byte, 32)
	_, err = rng.Read(salt)

	// Make DH private key
	privKey := diffieHellman.GeneratePrivateKey(len(grp.GetPBytes()), grp, rng)

	// make the ID
	newId, err := xx.NewID(rsaKey.GetPublic(), salt, id.User)
	if err != nil {
		return ReceptionIdentity{}, err
	}

	privKeyBytes, err := privKey.MarshalJSON()
	if err != nil {
		return ReceptionIdentity{}, err
	}

	grpBytes, err := grp.MarshalJSON()
	if err != nil {
		return ReceptionIdentity{}, err
	}

	// Create the identity object
	rsaPem := rsa.CreatePrivateKeyPem(rsaKey)
	I := ReceptionIdentity{
		ID:            newId,
		RSAPrivatePem: rsaPem,
		Salt:          salt,
		DHKeyPrivate:  privKeyBytes,
		E2eGrp:        grpBytes,
	}

	return I, nil
}

// MakeLegacyReceptionIdentity generates the cryptographic identity for
// receiving messages based on the extant stored user.Info.
func MakeLegacyReceptionIdentity(net *Cmix) (ReceptionIdentity, error) {
	userInfo := net.GetStorage().PortableUserInfo()
	return buildReceptionIdentity(userInfo.ReceptionID, userInfo.ReceptionSalt,
		userInfo.ReceptionRSA, net.GetStorage().GetE2EGroup(), userInfo.E2eDhPrivateKey)
}

// DeepCopy produces a safe copy of the ReceptionIdentity.
func (r ReceptionIdentity) DeepCopy() ReceptionIdentity {
	saltCopy := make([]byte, len(r.Salt))
	copy(saltCopy, r.Salt)

	dhKeyCopy := make([]byte, len(r.DHKeyPrivate))
	copy(dhKeyCopy, r.DHKeyPrivate)

	grpCopy := make([]byte, len(r.E2eGrp))
	copy(grpCopy, r.E2eGrp)
	return ReceptionIdentity{
		ID:            r.ID.DeepCopy(),
		RSAPrivatePem: r.RSAPrivatePem,
		Salt:          saltCopy,
		DHKeyPrivate:  dhKeyCopy,
		E2eGrp:        grpCopy,
	}
}

// GetContact returns a contact.Contact object of the reception identity.
func (r ReceptionIdentity) GetContact() contact.Contact {
	grp, _ := r.GetGroup()
	dhKeyPriv, _ := r.GetDHKeyPrivate()

	ct := contact.Contact{
		ID:             r.ID,
		DhPubKey:       diffieHellman.GeneratePublicKey(dhKeyPriv, grp),
		OwnershipProof: nil,
		Facts:          make([]fact.Fact, 0),
	}
	return ct
}

// buildReceptionIdentity creates a new ReceptionIdentity from the given
// user.Info.
func buildReceptionIdentity(receptionId *id.ID, receptionSalt []byte,
	receptionRsa *rsa.PrivateKey, e2eGrp *cyclic.Group, dHPrivkey *cyclic.Int) (
	ReceptionIdentity, error) {
	saltCopy := make([]byte, len(receptionSalt))
	copy(saltCopy, receptionSalt)

	grp, err := e2eGrp.MarshalJSON()
	if err != nil {
		return ReceptionIdentity{}, err
	}
	privKey, err := dHPrivkey.MarshalJSON()
	if err != nil {
		return ReceptionIdentity{}, err
	}

	return ReceptionIdentity{
		ID:            receptionId.DeepCopy(),
		RSAPrivatePem: rsa.CreatePrivateKeyPem(receptionRsa),
		Salt:          saltCopy,
		DHKeyPrivate:  privKey,
		E2eGrp:        grp,
	}, nil
}

// TransmissionIdentity represents the identity used to transmit over the
// network via a specific Cmix object.
type TransmissionIdentity struct {
	ID            *id.ID
	RSAPrivatePem *rsa.PrivateKey
	Salt          []byte

	// Timestamp of when the user has registered with the network
	RegistrationTimestamp int64
}

// DeepCopy produces a safe copy of a TransmissionIdentity.
func (t TransmissionIdentity) DeepCopy() TransmissionIdentity {
	saltCopy := make([]byte, len(t.Salt))
	copy(saltCopy, t.Salt)
	return TransmissionIdentity{
		ID:                    t.ID.DeepCopy(),
		RSAPrivatePem:         t.RSAPrivatePem,
		Salt:                  saltCopy,
		RegistrationTimestamp: t.RegistrationTimestamp,
	}
}

// buildTransmissionIdentity creates a new TransmissionIdentity from the given
// user.Info.
func buildTransmissionIdentity(userInfo user.Info) TransmissionIdentity {
	saltCopy := make([]byte, len(userInfo.TransmissionSalt))
	copy(saltCopy, userInfo.TransmissionSalt)
	return TransmissionIdentity{
		ID:                    userInfo.TransmissionID.DeepCopy(),
		RSAPrivatePem:         userInfo.TransmissionRSA,
		Salt:                  saltCopy,
		RegistrationTimestamp: userInfo.RegistrationTimestamp,
	}
}
