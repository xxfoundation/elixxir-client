///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Session object definition

package storage

import (
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/elixxir/client/storage/hostList"
	"gitlab.com/elixxir/client/storage/rounds"
	"gitlab.com/elixxir/client/storage/ud"
	"gitlab.com/xx_network/primitives/rateLimiting"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	userInterface "gitlab.com/elixxir/client/interfaces/user"
	"gitlab.com/elixxir/client/storage/auth"
	"gitlab.com/elixxir/client/storage/clientVersion"
	"gitlab.com/elixxir/client/storage/cmix"
	"gitlab.com/elixxir/client/storage/conversation"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/client/storage/partition"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

// Number of rounds to store in the CheckedRound buffer
const CheckRoundsMaxSize = 1000000 / 64
const currentSessionVersion = 0

// Session object, backed by encrypted filestore
type Session struct {
	kv *versioned.KV

	mux sync.RWMutex

	//memoized data
	regStatus RegistrationStatus
	ndf       *ndf.NetworkDefinition

	//sub-stores
	e2e                 *e2e.Store
	cmix                *cmix.Store
	user                *user.User
	conversations       *conversation.Store
	partition           *partition.Store
	auth                *auth.Store
	criticalMessages    *utility.E2eMessageBuffer
	criticalRawMessages *utility.CmixMessageBuffer
	bucketStore         *rateLimiting.Bucket
	bucketParamStore    *utility.BucketParamStore
	garbledMessages     *utility.MeteredCmixMessageBuffer
	reception           *reception.Store
	clientVersion       *clientVersion.Store
	uncheckedRounds     *rounds.UncheckedRoundStore
	hostList            *hostList.Store
	edgeCheck           *edge.Store
	ringBuff            *conversation.Buff
	ud                  *ud.Store
}

// Initialize a new Session object
func initStore(baseDir, password string) (*Session, error) {
	fs, err := ekv.NewFilestore(baseDir, password)
	var s *Session
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to create storage session")
	}

	s = &Session{
		kv: versioned.NewKV(fs),
	}

	return s, nil
}

// Creates new UserData in the session
func New(baseDir, password string, u userInterface.User,
	currentVersion version.Version, cmixGrp, e2eGrp *cyclic.Group,
	rng *fastRNG.StreamGenerator,
	rateLimitParams ndf.RateLimiting) (*Session, error) {

	s, err := initStore(baseDir, password)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to create session for %s", baseDir)
	}

	err = s.newRegStatus()
	if err != nil {
		return nil, errors.WithMessage(err,
			"Create new session")
	}

	s.user, err = user.NewUser(s.kv, u.TransmissionID, u.ReceptionID, u.TransmissionSalt,
		u.ReceptionSalt, u.TransmissionRSA, u.ReceptionRSA, u.Precanned)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create user")
	}
	uid := s.user.GetCryptographicIdentity().GetReceptionID()

	s.cmix, err = cmix.NewStore(cmixGrp, s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create cmix store")
	}

	s.e2e, err = e2e.NewStore(e2eGrp, s.kv, u.E2eDhPrivateKey, uid, rng)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create e2e store")
	}

	s.auth, err = auth.NewStore(s.kv, e2eGrp, []*cyclic.Int{u.E2eDhPrivateKey})
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create auth store")
	}

	s.garbledMessages, err = utility.NewMeteredCmixMessageBuffer(s.kv, garbledMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create garbledMessages buffer")
	}

	s.criticalMessages, err = utility.NewE2eMessageBuffer(s.kv, criticalMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create e2e critical message buffer")
	}

	s.criticalRawMessages, err = utility.NewCmixMessageBuffer(s.kv, criticalRawMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create raw critical message buffer")
	}

	s.conversations = conversation.NewStore(s.kv)
	s.partition = partition.New(s.kv)

	s.reception = reception.NewStore(s.kv)

	s.clientVersion, err = clientVersion.NewStore(currentVersion, s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create client version store.")
	}

	s.uncheckedRounds, err = rounds.NewUncheckedStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create unchecked round store")
	}

	s.hostList = hostList.NewStore(s.kv)

	s.edgeCheck, err = edge.NewStore(s.kv, u.ReceptionID)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create edge check store")
	}

	s.bucketParamStore, err = utility.NewBucketParamsStore(
		uint32(rateLimitParams.Capacity), uint32(rateLimitParams.LeakedTokens),
		time.Duration(rateLimitParams.LeakDuration), s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create bucket params store")
	}

	s.bucketStore = utility.NewStoredBucket(uint32(rateLimitParams.Capacity), uint32(rateLimitParams.LeakedTokens),
		time.Duration(rateLimitParams.LeakDuration), s.kv)

	s.ud, err = ud.NewStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create ud store")
	}

	return s, nil
}

// Loads existing user data into the session
func Load(baseDir, password string, currentVersion version.Version,
	rng *fastRNG.StreamGenerator) (*Session, error) {

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

	s.user, err = user.LoadUser(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.cmix, err = cmix.LoadStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	uid := s.user.GetCryptographicIdentity().GetReceptionID()

	s.e2e, err = e2e.LoadStore(s.kv, uid, rng)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.auth, err = auth.LoadStore(s.kv, s.e2e.GetGroup(),
		[]*cyclic.Int{s.e2e.GetDHPrivateKey()})
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load auth store")
	}

	s.criticalMessages, err = utility.LoadE2eMessageBuffer(s.kv, criticalMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load session")
	}

	s.criticalRawMessages, err = utility.LoadCmixMessageBuffer(s.kv, criticalRawMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load raw critical message buffer")
	}

	s.garbledMessages, err = utility.LoadMeteredCmixMessageBuffer(s.kv, garbledMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load session")
	}

	s.conversations = conversation.NewStore(s.kv)
	s.partition = partition.Load(s.kv)

	s.reception = reception.LoadStore(s.kv)

	s.uncheckedRounds, err = rounds.LoadUncheckedStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load unchecked round store")
	}

	s.hostList = hostList.NewStore(s.kv)

	s.edgeCheck, err = edge.LoadStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load edge check store")
	}

	s.bucketParamStore, err = utility.LoadBucketParamsStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load bucket params store")
	}

	params := s.bucketParamStore.Get()
	s.bucketStore, err = utility.LoadBucket(params.Capacity, params.LeakedTokens,
		params.LeakDuration, s.kv)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load bucket store")
	}

	s.ud, err = ud.NewOrLoadStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load ud store")
	}

	return s, nil
}

func (s *Session) User() *user.User {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.user
}

func (s *Session) Cmix() *cmix.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.cmix
}

func (s *Session) E2e() *e2e.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.e2e
}

func (s *Session) Auth() *auth.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.auth
}

func (s *Session) Reception() *reception.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.reception
}

func (s *Session) GetCriticalMessages() *utility.E2eMessageBuffer {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.criticalMessages
}

func (s *Session) GetCriticalRawMessages() *utility.CmixMessageBuffer {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.criticalRawMessages
}

func (s *Session) GetGarbledMessages() *utility.MeteredCmixMessageBuffer {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.garbledMessages
}

// GetClientVersion returns the version of the client storage.
func (s *Session) GetClientVersion() version.Version {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.clientVersion.Get()
}

func (s *Session) Conversations() *conversation.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.conversations
}

func (s *Session) Partition() *partition.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.partition
}

func (s *Session) UncheckedRounds() *rounds.UncheckedRoundStore {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.uncheckedRounds
}

func (s *Session) HostList() *hostList.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.hostList
}

// GetEdge returns the edge preimage store.
func (s *Session) GetEdge() *edge.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.edgeCheck
}

func (s *Session) GetUd() *ud.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.ud
}

// GetBucketParams returns the bucket params store.
func (s *Session) GetBucketParams() *utility.BucketParamStore {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.bucketParamStore
}

func (s *Session) GetBucket() *rateLimiting.Bucket {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.bucketStore
}

// Get an object from the session
func (s *Session) Get(key string) (*versioned.Object, error) {
	return s.kv.Get(key, currentSessionVersion)
}

// Set a value in the session
func (s *Session) Set(key string, object *versioned.Object) error {
	return s.kv.Set(key, currentSessionVersion, object)
}

// delete a value in the session
func (s *Session) Delete(key string) error {
	return s.kv.Delete(key, currentSessionVersion)
}

// GetKV returns the Session versioned.KV.
func (s *Session) GetKV() *versioned.KV {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.kv
}

// Initializes a Session object wrapped around a MemStore object.
// FOR TESTING ONLY
func InitTestingSession(i interface{}) *Session {
	switch i.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("InitTestingSession is restricted to testing only. Got %T", i)
	}

	privKey, _ := rsa.LoadPrivateKeyFromPem([]byte("-----BEGIN PRIVATE KEY-----\n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\nOaRA5ZbdE7g7vxKRV36jT3wvD7W+\n-----END PRIVATE KEY-----\n"))
	store := make(ekv.Memstore)
	kv := versioned.NewKV(store)
	s := &Session{kv: kv}
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

	s.user = u
	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString("9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48"+
			"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F"+
			"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5"+
			"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2"+
			"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41"+
			"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE"+
			"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15"+
			"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B", 16),
		large.NewIntFromString("5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613"+
			"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4"+
			"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472"+
			"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5"+
			"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA"+
			"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71"+
			"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0"+
			"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7", 16))
	cmixStore, err := cmix.NewStore(cmixGrp, kv)
	if err != nil {
		jww.FATAL.Panicf("InitTestingSession failed to create dummy cmix session: %+v", err)
	}
	s.cmix = cmixStore

	s.bucketParamStore, err = utility.NewBucketParamsStore(10, 11, 12, kv)
	if err != nil {
		jww.FATAL.Panicf("InitTestingSession failed to create NewBucketParamsStore session: %+v", err)
	}
	s.bucketStore = utility.NewStoredBucket(10, 11, 12, kv)

	e2eStore, err := e2e.NewStore(cmixGrp, kv, cmixGrp.NewInt(2), uid,
		fastRNG.NewStreamGenerator(7, 3, csprng.NewSystemRNG))
	if err != nil {
		jww.FATAL.Panicf("InitTestingSession failed to create dummy cmix session: %+v", err)
	}
	s.e2e = e2eStore

	s.criticalMessages, err = utility.NewE2eMessageBuffer(s.kv, criticalMessagesKey)
	if err != nil {
		jww.FATAL.Panicf("InitTestingSession failed to create dummy critical messages: %+v", err)
	}

	s.garbledMessages, err = utility.NewMeteredCmixMessageBuffer(s.kv, garbledMessagesKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to create garbledMessages buffer: %+v", err)
	}

	s.conversations = conversation.NewStore(s.kv)
	s.partition = partition.New(s.kv)

	s.reception = reception.NewStore(s.kv)

	s.uncheckedRounds, err = rounds.NewUncheckedStore(s.kv)
	if err != nil {
		jww.FATAL.Panicf("Failed to create uncheckRound store: %v", err)
	}

	s.hostList = hostList.NewStore(s.kv)

	privKeys := make([]*cyclic.Int, 10)
	pubKeys := make([]*cyclic.Int, 10)
	for i := range privKeys {
		privKeys[i] = cmixGrp.NewInt(5)
		pubKeys[i] = cmixGrp.ExpG(privKeys[i], cmixGrp.NewInt(1))
	}

	s.auth, err = auth.NewStore(s.kv, cmixGrp, privKeys)
	if err != nil {
		jww.FATAL.Panicf("Failed to create auth store: %v", err)
	}

	s.edgeCheck, err = edge.NewStore(s.kv, uid)
	if err != nil {
		jww.FATAL.Panicf("Failed to create new edge Store: %+v", err)
	}

	// todo: uncomment once NewBuff has been added properly
	//s.ringBuff, err = conversation.NewBuff(s.kv, 100)
	//if err != nil {
	//	jww.FATAL.Panicf("Failed to create ring buffer store: %+v", err)
	//}

	s.ud, err = ud.NewStore(s.kv)
	if err != nil {
		jww.FATAL.Panicf("Failed to create ud store: %v", err)
	}

	return s
}
