package channels

import (
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
)

var dummyNicknameUpdate = func(channelId *id.ID, nickname string, exists bool) {}

// Unit test. Tests that once you set a nickname with SetNickname, you can
// retrieve the nickname using GetNickname.
func TestNicknameManager_SetGetNickname(t *testing.T) {
	rkv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	rkv, err := rkv.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)
	nm := loadOrNewNicknameManager(rkv, dummyNicknameUpdate)

	for i := 0; i < numTests; i++ {
		chId := id.NewIdFromUInt(uint64(i), id.User, t)
		nickname := "nickname#" + strconv.Itoa(i)
		err := nm.SetNickname(nickname, chId)
		if err != nil {
			t.Fatalf("SetNickname error when setting %s: %+v", nickname, err)
		}

		received, _ := nm.GetNickname(chId)
		if received != nickname {
			t.Fatalf("GetNickname did not return expected values."+
				"\nExpected: %s"+
				"\nReceived: %s", nickname, received)
		}
	}
}

// Unit test. Tests that once you set a nickname with SetNickname, you can
// retrieve the nickname using GetNickname after a reload.
func TestNicknameManager_SetGetNickname_Reload(t *testing.T) {
	rkv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	rkv, err := rkv.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)
	nm := loadOrNewNicknameManager(rkv, dummyNicknameUpdate)

	for i := 0; i < numTests; i++ {
		chId := id.NewIdFromUInt(uint64(i), id.User, t)
		nickname := "nickname#" + strconv.Itoa(i)
		err := nm.SetNickname(nickname, chId)
		if err != nil {
			t.Fatalf("SetNickname error when setting %s: %+v", nickname, err)
		}
	}

	nm2 := loadOrNewNicknameManager(rkv, dummyNicknameUpdate)

	for i := 0; i < numTests; i++ {
		chId := id.NewIdFromUInt(uint64(i), id.User, t)
		nick, exists := nm2.GetNickname(chId)
		if !exists {
			t.Fatalf("Nickname %d not found  ", i)
		}
		expected := "nickname#" + strconv.Itoa(i)
		if nick != expected {
			t.Fatalf("Nickname %d not found, expected: %s, received: %s ",
				i, expected, nick)
		}
	}
}

// Error case: Tests that nicknameManager.GetNickname returns a false boolean
// if no nickname has been set with the channel ID.
func TestNicknameManager_GetNickname_Error(t *testing.T) {
	rkv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	rkv, err := rkv.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)
	nm := loadOrNewNicknameManager(rkv, dummyNicknameUpdate)

	for i := 0; i < numTests; i++ {
		chId := id.NewIdFromUInt(uint64(i), id.User, t)
		_, exists := nm.GetNickname(chId)
		if exists {
			t.Fatalf("GetNickname expected error case: " +
				"This should not retrieve nicknames for channel IDs " +
				"that are not set.")
		}
	}
}

func TestNicknameManager_DeleteNickname(t *testing.T) {
	kv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	kv, err := kv.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)
	nm := loadOrNewNicknameManager(kv, dummyNicknameUpdate)

	for i := 0; i < numTests; i++ {
		chId := id.NewIdFromUInt(uint64(i), id.User, t)
		nickname := "nickname#" + strconv.Itoa(i)
		err := nm.SetNickname(nickname, chId)
		if err != nil {
			t.Fatalf("SetNickname error when setting %s: %+v", nickname, err)
		}

		err = nm.DeleteNickname(chId)
		if err != nil {
			t.Fatalf("DeleteNickname error: %+v", err)
		}

		_, exists := nm.GetNickname(chId)
		if exists {
			t.Fatalf("GetNickname expected error case: " +
				"This should not retrieve nicknames for channel IDs " +
				"that are not set.")
		}
	}
}

func TestNicknameManager_mapUpdate(t *testing.T) {

	numIDs := 100

	wg := &sync.WaitGroup{}
	wg.Add(numIDs)
	expectedUpdates := make(map[id.ID]nicknameUpdate, numIDs)
	edits := make(map[string]versioned.ElementEdit, numIDs)

	rng := rand.New(rand.NewSource(69))

	// check that all callbacks get called correctly
	testingCB := func(channelId *id.ID, nickname string, exists bool) {
		receivedUpdate := nicknameUpdate{
			ChannelId:      channelId,
			Nickname:       nickname,
			NicknameExists: exists,
		}
		expectedNU, exists := expectedUpdates[*channelId]
		if !exists {
			t.Errorf("Update not found in list of updates fpr: %s", channelId)
		} else if !expectedNU.Equals(receivedUpdate) {
			t.Errorf("updates do not match: %+v vs %+v", receivedUpdate, expectedNU)
		}

		wg.Done()
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	kv, err := kv.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)
	nm := loadOrNewNicknameManager(kv, func(channelId *id.ID, nickname string, exists bool) {})

	// build the input and output data
	for i := 0; i < numIDs; i++ {

		cid := &id.ID{}
		cid[0] = byte(i)

		nicknameBytes := make([]byte, 10)
		rng.Read(nicknameBytes)
		nickname := base64.StdEncoding.EncodeToString(nicknameBytes)

		// make 1/3 chance it will be deleted
		existsChoice := make([]byte, 1)
		rng.Read(existsChoice)
		op := versioned.KeyOperation(int(existsChoice[0]) % 3)
		data, _ := json.Marshal(&nickname)

		nu := nicknameUpdate{
			ChannelId:      cid,
			Nickname:       nickname,
			NicknameExists: true,
		}

		if op == versioned.Deleted { // set the nickname if it is to be deleted so we can test deletion works
			if err := nm.SetNickname(nickname, cid); err != nil {
				t.Fatalf("Failed to set nickname for deletion: %+v", err)
			}
			data = nil
			nu.Nickname = ""
			nu.NicknameExists = false
		} else if op == versioned.Updated {
			rng.Read(nicknameBytes)
			nicknameOld := base64.StdEncoding.EncodeToString(nicknameBytes)
			if err := nm.SetNickname(nicknameOld, cid); err != nil {
				t.Fatalf("Failed to set nickname for updating: %+v", err)
			}
		}

		expectedUpdates[*cid] = nu

		//create the edit that will be processed

		edits[marshalChID(cid)] = versioned.ElementEdit{
			OldElement: nil,
			NewElement: &versioned.Object{
				Version:   0,
				Timestamp: time.Now(),
				Data:      data,
			},
			Operation: op,
		}
	}

	time.Sleep(1 * time.Second)

	nm.callback = testingCB

	nm.mapUpdate(edits)

	wg.Wait()

	//check that the local store is in the correct state

	for cID, update := range expectedUpdates {
		if !update.NicknameExists {
			_, exists := nm.GetNickname(&cID)
			if exists {
				t.Errorf("Nickname exists when it shouldt")
			}
		} else {
			nickname, exists := nm.GetNickname(&cID)
			if !exists {
				t.Errorf("Nickname does not exist when it should")
			} else {
				if nickname != update.Nickname {
					t.Errorf("Nickname is not correct")
				}
			}
		}
	}

}
