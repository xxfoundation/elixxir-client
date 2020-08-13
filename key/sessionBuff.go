package key

import (
	"github.com/pkg/errors"
	"sync"
)

type SessionBuff struct {
	sessions    []*Session
	sessionByID map[SessionID]*Session

	mux sync.RWMutex
}

type SessionBuffDisk struct {
	sessions []SessionID
}

func NewSessionBuff(n int, deletion func(session *Session)) *SessionBuff { return &SessionBuff{} }

//ekv functions
func (sb *SessionBuff) Marshal() ([]byte, error) { return nil, nil }
func (sb *SessionBuff) Unmarshal([]byte) error   { return nil }

func (sb *SessionBuff) AddSession(s *Session) {
	sb.mux.Lock()
	defer sb.mux.Unlock()

	sb.sessions = append([]*Session{s}, sb.sessions...)
	sb.sessionByID[s.GetID()] = s
	return
}

func (sb *SessionBuff) GetNewest() *Session {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	if len(sb.sessions) == 0 {
		return nil
	}
	return sb.sessions[0]
}

// returns the session which is most likely to be successful for sending
func (sb *SessionBuff) GetSessionForSending() *Session {
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

func (sb *SessionBuff) GetNewestConfirmed() *Session {
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

func (sb *SessionBuff) GetByID(id SessionID) *Session {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	return sb.sessionByID[id]
}

// sets the passed session ID as confirmed. Call "GetSessionRotation" after
// to get any sessions that are to be deleted and then "DeleteSession" to
// remove them
func (sb *SessionBuff) Confirm(id SessionID) error {
	sb.mux.Lock()
	defer sb.mux.Unlock()
	s, ok := sb.sessionByID[id]
	if !ok {
		return errors.Errorf("Could not confirm session %s, does not exist", s.GetID())
	}

	s.confirm()
	return nil
}

func (sb *SessionBuff) Clean() error {
}
}
//find the sessions position in the session buffer
loc := -1
for i, sBuf := range sb.sessions{
if sBuf==s{
loc = i
}
