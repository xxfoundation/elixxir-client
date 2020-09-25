////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"sync"
	"time"
)

const maxUnconfirmed uint = 3
const currentSessionBuffVersion = 0

type sessionBuff struct {
	manager *Manager

	kv *versioned.KV

	sessions    []*Session
	sessionByID map[SessionID]*Session

	key string

	mux     sync.RWMutex
	sendMux sync.Mutex
}

func NewSessionBuff(manager *Manager, key string) *sessionBuff {
	return &sessionBuff{
		manager:     manager,
		sessions:    make([]*Session, 0),
		sessionByID: make(map[SessionID]*Session),
		key:         key,
		kv:          manager.kv,
	}
}

func LoadSessionBuff(manager *Manager, key string) (*sessionBuff, error) {
	sb := &sessionBuff{
		manager:     manager,
		sessionByID: make(map[SessionID]*Session),
		key:         key,
		kv:          manager.kv,
	}

	key = makeSessionBuffKey(key)

	obj, err := manager.kv.Get(key)
	if err != nil {
		return nil, err
	}

	err = sb.unmarshal(obj.Data)

	if err != nil {
		return nil, err
	}

	return sb, nil
}

func (sb *sessionBuff) save() error {
	key := makeSessionBuffKey(sb.key)

	now := time.Now()

	data, err := sb.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentSessionBuffVersion,
		Timestamp: now,
		Data:      data,
	}

	return sb.kv.Set(key, &obj)
}

//ekv functions
func (sb *sessionBuff) marshal() ([]byte, error) {
	sessions := make([]SessionID, len(sb.sessions))

	index := 0
	for sid := range sb.sessionByID {
		sessions[index] = sid
		index++
	}

	return json.Marshal(&sessions)
}

func (sb *sessionBuff) unmarshal(b []byte) error {
	var sessions []SessionID

	err := json.Unmarshal(b, &sessions)

	if err != nil {
		return err
	}

	//load all the sessions
	for _, sid := range sessions {
		sessionKV := sb.kv.Prefix(makeSessionPrefix(sid))
		session, err := loadSession(sb.manager, sessionKV)
		if err != nil {
			jww.FATAL.Panicf("Failed to load session %s for %s: %s",
				makeSessionPrefix(sid), sb.manager.partner, err.Error())
		}
		sb.addSession(session)
	}

	return nil
}

func (sb *sessionBuff) AddSession(s *Session) {
	sb.mux.Lock()
	defer sb.mux.Unlock()

	sb.addSession(s)
	if err := sb.save(); err != nil {
		key := makeSessionBuffKey(sb.key)
		jww.FATAL.Printf("Failed to save Session Buffer %s after "+
			"adding session %s: %s", key, s, err)
	}

	return
}

func (sb *sessionBuff) addSession(s *Session) {
	sb.sessions = append([]*Session{s}, sb.sessions...)
	sb.sessionByID[s.GetID()] = s
	return
}

func (sb *sessionBuff) GetNewest() *Session {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	if len(sb.sessions) == 0 {
		return nil
	}
	return sb.sessions[0]
}

// returns the key  which is most likely to be successful for sending
func (sb *sessionBuff) getKeyForSending() (*Key, error) {
	sb.sendMux.Lock()
	defer sb.sendMux.Unlock()
	s := sb.getSessionForSending()
	if s == nil {
		return nil, errors.New("Failed to find a session for sending")
	}

	return s.PopKey()
}

// returns the session which is most likely to be successful for sending
func (sb *sessionBuff) getSessionForSending() *Session {
	sessions := sb.sessions

	var confirmedRekey []*Session
	var unconfirmedActive []*Session
	var unconfirmedRekey []*Session

	for _, s := range sessions {
		status := s.Status()
		confirmed := s.IsConfirmed()
		if status == Active && confirmed {
			//always return the first confirmed active, happy path
			return s
		} else if status == RekeyNeeded && confirmed {
			confirmedRekey = append(confirmedRekey, s)
		} else if status == Active && !confirmed {
			unconfirmedActive = append(unconfirmedActive, s)
		} else if status == RekeyNeeded && !confirmed {
			unconfirmedRekey = append(unconfirmedRekey, s)
		}
	}

	//return the newest based upon priority
	if len(confirmedRekey) > 0 {
		return confirmedRekey[0]
	} else if len(unconfirmedActive) > 0 {
		return unconfirmedActive[0]
	} else if len(unconfirmedRekey) > 0 {
		return unconfirmedRekey[0]
	}

	return nil
}

// returns a list of session that need rekeys. Nil instances mean a new rekey
// from scratch
func (sb *sessionBuff) TriggerNegotiation() []*Session {
	//dont need to take the lock due to the use of a copy of the buffer
	sessions := sb.getInternalBufferShallowCopy()
	var instructions []*Session
	for _, ses := range sessions {
		if ses.triggerNegotiation() {
			instructions = append(instructions, ses)
		}
	}
	return instructions
}

// returns a key which should be used for rekeying
func (sb *sessionBuff) getKeyForRekey() (*Key, error) {
	sb.sendMux.Lock()
	defer sb.sendMux.Unlock()
	s := sb.getNewestRekeyableSession()
	if s == nil {
		return nil, errors.New("Failed to find a session for rekeying")
	}

	return s.PopReKey()
}

// returns the newest session which can be used to start a key negotiation
func (sb *sessionBuff) getNewestRekeyableSession() *Session {
	//dont need to take the lock due to the use of a copy of the buffer
	sessions := sb.getInternalBufferShallowCopy()

	if len(sessions) == 0 {
		return nil
	}

	for _, s := range sb.sessions {
		// This looks like it might not be thread safe, I think it is because
		// the failure mode is it skips to a lower key to rekey with, which is
		// always valid. It isn't clear it can fail though because we are
		// accessing the data in the same order it would be written (i think)
		if s.Status() != RekeyEmpty && s.IsConfirmed() {
			return s
		}
	}
	return nil
}

func (sb *sessionBuff) GetByID(id SessionID) *Session {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	return sb.sessionByID[id]
}

// sets the passed session ID as confirmed. Call "GetSessionRotation" after
// to get any sessions that are to be deleted and then "DeleteSession" to
// remove them
func (sb *sessionBuff) Confirm(id SessionID) error {
	sb.mux.Lock()
	defer sb.mux.Unlock()
	fmt.Printf("sb: %v\n", sb)
	fmt.Printf("sb.sessionById: %v\n", sb.sessionByID)

	s, ok := sb.sessionByID[id]
	if !ok {
		return errors.Errorf("Could not confirm session %s, does not exist", id)
	}

	s.SetNegotiationStatus(Confirmed)

	sb.clean()

	return nil
}

// adding or removing a session is always done via replacing the entire
// slice, this allow us to copy the slice under the read lock and do the
// rest of the work while not taking the lock
func (sb *sessionBuff) getInternalBufferShallowCopy() []*Session {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	return sb.sessions
}

func (sb *sessionBuff) clean() {

	numConfirmed := uint(0)

	var newSessions []*Session
	editsMade := false

	for _, s := range sb.sessions {
		if s.IsConfirmed() {
			numConfirmed++
			//if the number of newer confirmed is sufficient, delete the confirmed
			if numConfirmed > maxUnconfirmed {
				delete(sb.sessionByID, s.GetID())
				s.Delete()
				editsMade = true
				continue
			}
		}
		newSessions = append(newSessions, s)
	}

	//only do the update and save if changes occured
	if editsMade {
		sb.sessions = newSessions

		if err := sb.save(); err != nil {
			key := makeSessionBuffKey(sb.key)
			jww.FATAL.Printf("Failed to save Session Buffer %s after "+
				"clean: %s", key, err)
		}
	}
}

func makeSessionBuffKey(key string) string {
	return "sessionBuffer" + key
}
