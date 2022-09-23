package channels

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"testing"
)

// Unit test. Tests that once you set a nickname with SetNickname, you can
// retrieve the nickname using GetNickname.
func TestNicknameManager_SetGetNickname(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	nm := loadOrNewNicknameManager(kv)

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

// Error case: Tests that nicknameManager.GetNickname returns a false boolean
// if no nickname has been set with the channel ID.
func TestNicknameManager_GetNickname_Error(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	nm := loadOrNewNicknameManager(kv)

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

// Unit test. Check that once you SetNickname and DeleteNickname,
// GetNickname returns a false boolean.
func TestNicknameManager_DeleteNickname(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	nm := loadOrNewNicknameManager(kv)

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
