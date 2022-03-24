///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Session object definition

package storage

import (
	"gitlab.com/elixxir/client/storage/utility"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	userInterface "gitlab.com/elixxir/client/interfaces/user"
	"gitlab.com/elixxir/client/storage/clientVersion"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

// Number of rounds to store in the CheckedRound buffer
const CheckRoundsMaxSize = 1000000 / 64
const currentSessionVersion = 0
const cmixGroupKey = "cmixGroup"
const e2eGroupKey = "e2eGroup"

// Session object, backed by encrypted filestore
type Session interface {
	GetClientVersion() version.Version
	Get(key string) (*versioned.Object, error)
	Set(key string, object *versioned.Object) error
	Delete(key string) error
	GetKV() *versioned.KV
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
	GetReceptionRSA() *rsa.PrivateKey
	GetTransmissionRSA() *rsa.PrivateKey
	IsPrecanned() bool
	SetUsername(username string) error
	GetUsername() (string, error)
	PortableUserInfo() userInterface.Info
	GetTransmissionRegistrationValidationSignature() []byte
	GetReceptionRegistrationValidationSignature() []byte
	GetRegistrationTimestamp() time.Time
	SetTransmissionRegistrationValidationSignature(b []byte)
	SetReceptionRegistrationValidationSignature(b []byte)
	SetRegistrationTimestamp(tsNano int64)
}

type session struct {
	kv *versioned.KV

	//memoized data
	mux       sync.RWMutex
	regStatus RegistrationStatus
	ndf       *ndf.NetworkDefinition

	//network parameters
	cmixGroup *cyclic.Group
	e2eGroup  *cyclic.Group

	//sub-stores
	*user.User
	clientVersion *clientVersion.Store
}

// Initialize a new Session object
func initStore(baseDir, password string) (*session, error) {
	fs, err := ekv.NewFilestore(baseDir, password)
	var s *session
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to create storage session")
	}

	s = &session{
		kv: versioned.NewKV(fs),
	}

	return s, nil
}

// Creates new UserData in the session
func New(baseDir, password string, u userInterface.Info,
	currentVersion version.Version, cmixGrp, e2eGrp *cyclic.Group) (Session, error) {

	s, err := initStore(baseDir, password)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to create session for %s", baseDir)
	}

	err = s.newRegStatus()
	if err != nil {
		return nil, errors.WithMessage(err,
			"Create new session")
	}

	s.User, err = user.NewUser(s.kv, u.TransmissionID, u.ReceptionID, u.TransmissionSalt,
		u.ReceptionSalt, u.TransmissionRSA, u.ReceptionRSA, u.Precanned)
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

// Loads existing user data into the session
func Load(baseDir, password string, currentVersion version.Version) (*Session, error) {

	s, err := initStore(baseDir, password)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	err = s.loadRegStatus()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.clientVersion, err = clientVersion.LoadStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load client version store.")
	}

	// Determine if the storage needs to be updated to the current version
	_, _, err = s.clientVersion.CheckUpdateRequired(currentVersion)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load client version store.")
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

// Set a value in the session
func (s *session) Set(key string, object *versioned.Object) error {
	return s.kv.Set(key, currentSessionVersion, object)
}

// delete a value in the session
func (s *session) Delete(key string) error {
	return s.kv.Delete(key, currentSessionVersion)
}

// GetKV returns the Session versioned.KV.
func (s *session) GetKV() *versioned.KV {
	return s.kv
}

// GetCmixGrouo returns cMix Group
func (s *session) GetCmixGroup() *cyclic.Group {
	return s.cmixGroup
}

// GetE2EGrouo returns cMix Group
func (s *session) GetE2EGroup() *cyclic.Group {
	return s.e2eGroup
}

// Initializes a Session object wrapped around a MemStore object.
// FOR TESTING ONLY
func InitTestingSession(i interface{}) Session {
	switch i.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("InitTestingSession is restricted to testing only. Got %T", i)
	}

	privKey, _ := rsa.LoadPrivateKeyFromPem([]byte("-----BEGIN PRIVATE KEY-----\n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\nOaRA5ZbdE7g7vxKRV36jT3wvD7W+\n-----END PRIVATE KEY-----\n"))
	store := make(ekv.Memstore)
	kv := versioned.NewKV(store)
	s := &session{kv: kv}
	uid := id.NewIdFromString("zezima", id.User, i)
	u, err := user.NewUser(kv, uid, uid, []byte("salt"), []byte("salt"), privKey, privKey, false)
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

	return s
}
