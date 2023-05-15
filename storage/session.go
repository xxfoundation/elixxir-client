////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Session object definition

package storage

import (
	"gitlab.com/elixxir/client/v4/collective"
	"math/rand"
	"sync"
	"testing"
	"time"

	"gitlab.com/elixxir/crypto/diffieHellman"

	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/xx_network/crypto/large"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/clientVersion"
	"gitlab.com/elixxir/client/v4/storage/user"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

const currentSessionVersion = 0

// NOTE: These are set this way for legacy purposes. If you want to change them
// you will need to set up and upgrade path for old session files
const cmixGroupKey = "cmix/GroupKey"
const e2eGroupKey = "e2eSession/Group"

// Session object, backed by encrypted versioned.KVc
type Session interface {
	GetClientVersion() version.Version
	Get(key string) (*versioned.Object, error)
	Set(key string, object *versioned.Object) error
	Delete(key string) error
	GetKV() versioned.KV
	GetCmixGroup() *cyclic.Group
	GetE2EGroup() *cyclic.Group
	ForwardRegistrationStatus(regStatus RegistrationStatus) error
	GetRegistrationStatus() RegistrationStatus
	SetRegCode(regCode string)
	GetRegCode() (string, error)
	SetNDF(def *ndf.NetworkDefinition)
	GetNDF() *ndf.NetworkDefinition
	GetTransmissionID() *id.ID
	GetTransmissionSalt() []byte
	GetReceptionID() *id.ID
	GetReceptionSalt() []byte
	GetReceptionRSA() rsa.PrivateKey
	GetTransmissionRSA() rsa.PrivateKey
	IsPrecanned() bool
	SetUsername(username string) error
	GetUsername() (string, error)
	PortableUserInfo() user.Info
	GetTransmissionRegistrationValidationSignature() []byte
	GetReceptionRegistrationValidationSignature() []byte
	GetRegistrationTimestamp() time.Time
	SetTransmissionRegistrationValidationSignature(b []byte)
	SetReceptionRegistrationValidationSignature(b []byte)
	SetRegistrationTimestamp(tsNano int64)
}

type session struct {
	kv versioned.KV

	// memoized data
	mux       sync.RWMutex
	regStatus RegistrationStatus
	ndf       *ndf.NetworkDefinition

	// network parameters
	cmixGroup *cyclic.Group
	e2eGroup  *cyclic.Group

	// sub-stores
	*user.User
	clientVersion *clientVersion.Store
}

// New UserData in the session
func New(storage versioned.KV, u user.Info,
	currentVersion version.Version,
	cmixGrp, e2eGrp *cyclic.Group) (Session, error) {

	s := &session{
		kv: storage,
	}

	err := s.newRegStatus()
	if err != nil {
		return nil, errors.WithMessage(err,
			"Create new session")
	}

	s.User, err = user.NewUser(s.kv, u.TransmissionID, u.ReceptionID,
		u.TransmissionSalt, u.ReceptionSalt, u.TransmissionRSA,
		u.ReceptionRSA, u.Precanned, u.E2eDhPrivateKey,
		u.E2eDhPublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create user")
	}

	s.clientVersion, err = clientVersion.NewStore(currentVersion, s.kv)

	if err = utility.StoreGroup(s.kv, cmixGrp, cmixGroupKey); err != nil {
		return nil, err
	}

	if err = utility.StoreGroup(s.kv, e2eGrp, e2eGroupKey); err != nil {
		return nil, err
	}

	s.cmixGroup = cmixGrp
	s.e2eGroup = e2eGrp
	return s, nil
}

// Load existing user data into the session
func Load(storage versioned.KV,
	currentVersion version.Version) (Session, error) {

	s := &session{
		kv: storage,
	}

	err := s.loadRegStatus()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.clientVersion, err = clientVersion.LoadStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load client version store.")
	}

	// Determine if the storage needs to be updated to the current version
	_, _, err = s.clientVersion.CheckUpdateRequired(currentVersion)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load client version store.")
	}

	s.User, err = user.LoadUser(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.cmixGroup, err = utility.LoadGroup(s.kv, cmixGroupKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.e2eGroup, err = utility.LoadGroup(s.kv, e2eGroupKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	return s, nil
}

// GetClientVersion returns the version of the client storage.
func (s *session) GetClientVersion() version.Version {
	return s.clientVersion.Get()
}

// Get an object from the session
func (s *session) Get(key string) (*versioned.Object, error) {
	return s.kv.Get(key, currentSessionVersion)
}

// Set a value in the session. If you wish to maintain versioning,
// the [versioned.Object]'s Version field must be set.
func (s *session) Set(key string, object *versioned.Object) error {
	return s.kv.Set(key, object)
}

// Delete a value in the session
func (s *session) Delete(key string) error {
	return s.kv.Delete(key, currentSessionVersion)
}

// GetKV returns the Session versioned.KV.
func (s *session) GetKV() versioned.KV {
	return s.kv
}

// GetCmixGroup returns cMix Group
func (s *session) GetCmixGroup() *cyclic.Group {
	return s.cmixGroup
}

// GetE2EGroup returns cMix Group
func (s *session) GetE2EGroup() *cyclic.Group {
	return s.e2eGroup
}

// InitTestingSession object wrapped around a MemStore object.
// FOR TESTING ONLY
func InitTestingSession(i interface{}) Session {
	switch i.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("InitTestingSession is restricted to testing only. Got %T", i)
	}
	sch := rsa.GetScheme()
	privKey, _ := sch.UnmarshalPrivateKeyPEM([]byte("-----BEGIN PRIVATE KEY-----\n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\nOaRA5ZbdE7g7vxKRV36jT3wvD7W+\n-----END PRIVATE KEY-----\n"))
	kv := collective.TestingKV(i, ekv.MakeMemstore(), collective.StandardPrefexs)
	s := &session{kv: kv}
	uid := id.NewIdFromString("zezima", id.User, i)

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	u, err := user.NewUser(kv, uid, uid, []byte("salt"), []byte("salt"), privKey, privKey, false, dhPrivKey, dhPubKey)
	if err != nil {
		jww.FATAL.Panicf("InitTestingSession failed to create dummy user: %+v", err)
	}
	u.SetTransmissionRegistrationValidationSignature([]byte("sig"))
	u.SetReceptionRegistrationValidationSignature([]byte("sig"))
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		jww.FATAL.Panicf("Could not parse precanned time: %v", err.Error())
	}
	u.SetRegistrationTimestamp(testTime.UnixNano())

	s.User = u
	s.cmixGroup = cyclic.NewGroup(
		large.NewIntFromString("9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642"+
			"F0B5C48C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757"+
			"264E5A1A44FFE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F"+
			"9716BFE6117C6B5B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091E"+
			"B51743BF33050C38DE235567E1B34C3D6A5C0CEAA1A0F368213C3D19843D"+
			"0B4B09DCB9FC72D39C8DE41F1BF14D4BB4563CA28371621CAD3324B6A2D3"+
			"92145BEBFAC748805236F5CA2FE92B871CD8F9C36D3292B5509CA8CAA77A"+
			"2ADFC7BFD77DDA6F71125A7456FEA153E433256A2261C6A06ED3693797E7"+
			"995FAD5AABBCFBE3EDA2741E375404AE25B", 16),
		large.NewIntFromString("5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E2480"+
			"9670716C613D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D"+
			"1AA58C4328A06C46A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A33"+
			"8661D10461C0D135472085057F3494309FFA73C611F78B32ADBB5740C361"+
			"C9F35BE90997DB2014E2EF5AA61782F52ABEB8BD6432C4DD097BC5423B28"+
			"5DAFB60DC364E8161F4A2A35ACA3A10B1C4D203CC76A470A33AFDCBDD929"+
			"59859ABD8B56E1725252D78EAC66E71BA9AE3F1DD2487199874393CD4D83"+
			"2186800654760E1E34C09E4D155179F9EC0DC4473F996BDCE6EED1CABED8"+
			"B6F116F7AD9CF505DF0F998E34AB27514B0FFE7", 16),
	)

	return s
}
