package reception

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/crypto/randomness"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"io"
	"math/big"
	"strconv"
	"sync"
	"time"
)

const receptionPrefix = "reception"
const receptionStoreStorageKey = "receptionStoreKey"
const receptionStoreStorageVersion = 0
const receptionIDSizeStorageKey = "receptionIDSizeKey"
const receptionIDSizeStorageVersion = 0
const defaultIDSize = 12

type Store struct{
	// identities which are being actively checked
	active 		[]*registration
	idSize 	    int

	kv *versioned.KV

	mux sync.Mutex
}

type storedReference struct {
	Eph    ephemeral.Id
	Source *id.ID
}

//creates a new reception store.  It starts empty
func NewStore(kv *versioned.KV)*Store{
	kv = kv.Prefix(receptionPrefix)
	s := &Store{
		active: make([]*registration, 0),
		idSize: defaultIDSize,
		kv:     kv,
	}

	//store the empty list
	if err := s.save(); err!=nil{
		jww.FATAL.Panicf("Failed to save new reception store: %+v", err)
	}

	//update the size so queries can be made
	s.UpdateIDSize(defaultIDSize)

	return s
}

func LoadStore(kv *versioned.KV)*Store{
	kv = kv.Prefix(receptionPrefix)
	s := &Store{
		kv:     kv,
	}

	// Load the versioned object for the reception list
	vo, err := kv.Get(receptionStoreStorageKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to get the reception storage list: %+v",
			err)
	}

	identities := make([]storedReference, len(s.active))
	err = json.Unmarshal(vo.Data, &identities)
	if err!=nil{
		jww.FATAL.Panicf("Failed to unmarshal the reception storage " +
			"list: %+v", err)
	}

	s.active = make([]*registration, len(identities))
	for i, sr := range identities{
		s.active[i], err = loadRegistration(sr.Eph, sr.Source, s.kv)
		if err!=nil{
			jww.FATAL.Panicf("Failed to load registration for %s: %+v",
				regPrefix(sr.Eph, sr.Source), err)
		}
	}

	//load the ephmemeral ID length
	vo, err = kv.Get(receptionIDSizeStorageKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to get the reception id size: %+v",
			err)
	}

	if s.idSize, err = strconv.Atoi(string(vo.Data)); err!=nil{
		jww.FATAL.Panicf("Failed to unmarshal the reception id size: %+v",
			err)
	}

	return s
}

func (s *Store)	save()error{
	identities := make([]storedReference, len(s.active))
	i := 0
	for _, reg := range s.active{
		if !reg.Ephemeral{
			identities[i] = storedReference{
				Eph:    reg.EphId,
				Source: reg.Source,
			}
			i++
		}
	}
	identities = identities[:i]

	data, err := json.Marshal(&identities)
	if err!=nil{
		return errors.WithMessage(err, "failed to store reception " +
			"store")
	}

	// Create versioned object with data
	obj := &versioned.Object{
		Version:   receptionStoreStorageVersion,
		Timestamp: time.Now(),
		Data:      data,
	}

	err = s.kv.Set(receptionStoreStorageKey, obj)
	if err!=nil{
		return errors.WithMessage(err, "Failed to store reception store")
	}

	return nil
}

func (s *Store)GetIdentity(rng io.Reader)(IdentityUse, error){
	s.mux.Lock()
	defer s.mux.Unlock()

	now := time.Now()

	//remove any now expired identities
	s.prune(now)

	var identity IdentityUse
	var err error

	// if the list is empty, we return a randomly generated identity to poll
	// with so we can continue tracking the network and to further obfuscate
	// network identities
	if len(s.active)==0{
		identity, err = generateFakeIdentity(rng, uint(s.idSize))
		if err!=nil{
			jww.FATAL.Panicf("Failed to generate a new ID when none " +
				"available: %+v", err)
		}
	}else{
		identity, err = s.selectIdentity(rng, now)
		if err!=nil{
			jww.FATAL.Panicf("Failed to select an id: %+v", err)
		}
	}

	//calculate the sampling period
	identity, err = identity.SetSamplingPeriod(rng)
	if err!=nil{
		jww.FATAL.Panicf("Failed to caluclate the sampling period: " +
			"%+v", err)
	}

	return identity, nil
}

func (s *Store)AddIdentity(identity Identity)error {
	s.mux.Lock()
	defer s.mux.Unlock()

	reg, err := newRegistration(identity, s.kv)
	if err!=nil{
		return errors.WithMessage(err,"failed to add new identity to " +
			"reception store")
	}

	s.active = append(s.active, reg)
	if !identity.Ephemeral{
		if err := s.save(); err!=nil{
			jww.FATAL.Panicf("Failed to save reception store after identity " +
				"addition")
		}
	}

	return nil
}

func (s *Store)RemoveIdentity(ephID ephemeral.Id)bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	for i:=0;i<len(s.active);i++{
		inQuestion := s.active[i]
		if bytes.Equal(inQuestion.EphId[:],ephID[:]){
			s.active = append(s.active[:i], s.active[i+1:]...)
			err := inQuestion.Delete()
			if err!=nil{
				jww.FATAL.Panicf("Failed to delete identity %s")
			}
			if !inQuestion.Ephemeral{
				if err := s.save(); err!=nil{
					jww.FATAL.Panicf("Failed to save reception store after " +
						"identity removal")
				}
			}

			return true
		}
	}

	return false
}

func (s *Store)UpdateIDSize(idSize uint){
	s.mux.Lock()
	defer s.mux.Unlock()
	s.idSize = int(idSize)
	//store the id size
	obj := &versioned.Object{
		Version:   receptionIDSizeStorageVersion,
		Timestamp: time.Now(),
		Data: []byte(strconv.Itoa(s.idSize)),
	}

	err := s.kv.Set(receptionIDSizeStorageKey, obj)
	if err!=nil{
		jww.FATAL.Panicf("Failed to store reception ID size: %+v", err)
	}
}

func (s *Store)prune(now time.Time) {
	lengthBefore := len(s.active)

	//prune the list
	for i:=0;i<len(s.active);i++{
		inQuestion := s.active[i]
		if now.After(inQuestion.End) && inQuestion.ExtraChecks ==0{
			if err := inQuestion.Delete(); err!=nil{
				jww.ERROR.Printf("Failed to delete Identity for %s: " +
					"%+v", inQuestion, err)
			}

			s.active = append(s.active[:i-1], s.active[i:]...)

			i--
		}
	}

	//save the list if it changed
	if lengthBefore!=len(s.active){
		if err := s.save(); err!=nil{
			jww.FATAL.Panicf("Failed to store reception storage")
		}
	}
}

func (s *Store)selectIdentity(rng io.Reader, now time.Time)(IdentityUse, error) {

	//choose a member from the list
	var selected *registration

	if len(s.active)==1{
		selected= s.active[0]
	}else{

		seed := make([]byte,32)
		if _, err := rng.Read(seed);err!=nil{
			return IdentityUse{}, errors.WithMessage(err, "Failed to " +
				"choose id due to rng failure")
		}

		h, err := hash.NewCMixHash()
		if err==nil{
			return IdentityUse{}, err
		}

		selectedNum := randomness.RandInInterval(
			big.NewInt(int64(len(s.active)-1)),seed,h)
		selected = s.active[selectedNum.Uint64()]
	}

	if now.After(selected.End){
		selected.ExtraChecks--
	}

	return IdentityUse{
		Identity:     selected.Identity,
		Fake:         false,
		KR:           selected.getKR(),
	}, nil
}