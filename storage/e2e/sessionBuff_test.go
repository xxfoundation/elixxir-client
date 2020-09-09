package e2e

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Shows that LoadSessionBuff returns an equivalent session buff to the one that was saved
func TestLoadSessionBuff(t *testing.T) {
	testBuff := makeTestSessionBuff(t)
	testBuff.manager.partner = id.NewIdFromUInt(5, id.User, t)
	session, _ := makeTestSession(t)
	testBuff.sessions = append(testBuff.sessions, session)
	err := testBuff.save()
	if err != nil {
		t.Fatal(err)
	}
	loadedBuff, err := LoadSessionBuff(testBuff.manager, "test", testBuff.manager.partner)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(testBuff, loadedBuff) {
		t.Error("buffers differed")
	}
}

// Shows that NewSessionBuff returns a valid session buff
func TestNewSessionBuff(t *testing.T) {

}

// Shows that AddSession adds one session to the buff
func TestSessionBuff_AddSession(t *testing.T) {

}

// Shows that Confirm confirms one session in the buff
func TestSessionBuff_Confirm(t *testing.T) {

}

func TestSessionBuff_GetByID(t *testing.T) {

}

func TestSessionBuff_GetNewest(t *testing.T) {

}

func TestSessionBuff_GetNewestRekeyableSession(t *testing.T) {

}

func TestSessionBuff_GetSessionForSending(t *testing.T) {

}

func TestSessionBuff_TriggerNegotiation(t *testing.T) {

}

// Make an empty session buff for testing
func makeTestSessionBuff(t *testing.T) *sessionBuff {
	grp := getGroup()

	//create context objects for general use
	fps := newFingerprints()
	ctx := &context{
		fa:  &fps,
		grp: grp,
		kv:  versioned.NewKV(make(ekv.Memstore)),
	}

	buff := &sessionBuff{
		manager: &Manager{
			ctx: ctx,
		},
		sessions:    make([]*Session, 0),
		sessionByID: make(map[SessionID]*Session),
		keyPrefix:   "test",
	}
	return buff
}
