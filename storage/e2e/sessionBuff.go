package e2e

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const maxUnconfirmed uint = 3
const currentSessionBuffVersion = 0

type sessionBuff struct {
	manager *Manager

	sessions    []*Session
	sessionByID map[SessionID]*Session

	keyPrefix string

	mux sync.RWMutex
}

func NewSessionBuff(manager *Manager, keyPrefix string) *sessionBuff {
	return &sessionBuff{
		manager:     manager,
		sessions:    make([]*Session, 0),
		sessionByID: make(map[SessionID]*Session),
		mux:         sync.RWMutex{},
		keyPrefix:   keyPrefix,
	}
}

func LoadSessionBuff(manager *Manager, keyPrefix string, partnerID *id.ID) (*sessionBuff, error) {
	sb := &sessionBuff{
		manager:     manager,
		sessionByID: make(map[SessionID]*Session),
		mux:         sync.RWMutex{},
	}

	key := makeSessionBuffKey(keyPrefix, partnerID)

	obj, err := manager.ctx.kv.Get(key)
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
	key := makeSessionBuffKey(sb.keyPrefix, sb.manager.partner)

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

	return sb.manager.ctx.kv.Set(key, &obj)
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

	sb.sessions = make([]*Session, len(sessions))

	//load all the sessions
	for _, sid := range sessions {
		key := makeSessionKey(sid)
		session, err := loadSession(sb.manager, key)
		if err != nil {
			jww.FATAL.Panicf("Failed to load session %s for %s: %s",
				key, sb.manager.partner, err.Error())
		}
		sb.addSession(session)
	}

	return nil
}

func (sb *sessionBuff) AddSession(s *Session) error {
	sb.mux.Lock()
	defer sb.mux.Unlock()

	sb.addSession(s)
	return sb.save()
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

// returns the session which is most likely to be successful for sending
func (sb *sessionBuff) GetSessionForSending() *Session {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	if len(sb.sessions) == 0 {
		return nil
	}

	var confirmedRekey []*Session
	var unconfirmedActive []*Session
	var unconfirmedRekey []*Session

	for _, s := range sb.sessions {
		status := s.Status()
		confirmed := s.confirmed
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

func (sb *sessionBuff) GetNewestConfirmed() *Session {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	if len(sb.sessions) == 0 {
		return nil
	}

	for _, s := range sb.sessions {
		status := s.Status()
		confirmed := s.confirmed
		if status != RekeyEmpty && confirmed {
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
	s, ok := sb.sessionByID[id]
	if !ok {
		return errors.Errorf("Could not confirm session %s, does not exist", s.GetID())
	}

	err := s.confirm()
	if err != nil {
		jww.FATAL.Panicf("Failed to confirm session "+
			"%s for %s: %s", s.GetID(), sb.manager.partner, err.Error())
	}

	return sb.clean()
}

func (sb *sessionBuff) clean() error {

	numConfirmed := uint(0)

	var newSessions []*Session

	for _, s := range sb.sessions {
		if s.IsConfirmed() {
			numConfirmed++
			//if the number of newer confirmed is sufficient, delete the confirmed
			if numConfirmed > maxUnconfirmed {
				delete(sb.sessionByID, s.GetID())
				err := s.Delete()
				if err != nil {
					jww.FATAL.Panicf("Failed to delete session store "+
						"%s for %s: %s", s.GetID(), sb.manager.partner, err.Error())
				}

				break
			}
		}
		newSessions = append(newSessions, s)
	}

	sb.sessions = newSessions

	return sb.save()
}

func makeSessionBuffKey(keyPrefix string, partnerID *id.ID) string {
	return keyPrefix + "sessionBuffer" + base64.StdEncoding.EncodeToString(partnerID.Marshal())
}