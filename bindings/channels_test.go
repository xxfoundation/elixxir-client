package bindings

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/v4/channels"
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
)

// Produces example JSON of NickNameUpdateJSON to be used for documentation.
func Test_BuildJSON_NickNameUpdateJSON(t *testing.T) {
	rng := rand.New(rand.NewSource(204103639))

	channelID, _ := id.NewRandomID(rng, id.User)

	jsonable := NickNameUpdateJSON{
		ChannelId: channelID,
		Nickname:  "billNyeTheScienceGuy",
		Exists:    true,
	}

	data, err := json.MarshalIndent(jsonable, "//  ", "  ")
	if err != nil {
		t.Errorf("Failed to JSON %T: %+v", jsonable, err)
	} else {
		fmt.Printf("//  %s\n", data)
	}
}

// Produces example JSON of NotificationUpdateJSON to be used for documentation.
func Test_BuildJSON_NotificationUpdateJSON(t *testing.T) {
	rng := rand.New(rand.NewSource(475107990))

	// Notifications update
	types := []channels.MessageType{channels.Text, channels.Pinned}
	levels := []channels.NotificationLevel{channels.NotifyPing, channels.NotifyAll}
	nuj := NotificationUpdateJSON{
		NotificationFilters:       []channels.NotificationFilter{},
		ChangedNotificationStates: []channels.NotificationState{},
		DeletedNotificationStates: []*id.ID{},
		MaxState:                  0,
	}

	var notificationLevelAllowLists = map[string]map[channels.NotificationLevel]channels.AllowLists{
		"symmetric": {
			channels.NotifyPing: {
				map[channels.MessageType]struct{}{channels.Text: {}, channels.FileTransfer: {}},
				map[channels.MessageType]struct{}{},
			},
			channels.NotifyAll: {
				map[channels.MessageType]struct{}{},
				map[channels.MessageType]struct{}{channels.Text: {}, channels.FileTransfer: {}},
			},
		},
		"asymmetric": {
			channels.NotifyPing: {
				map[channels.MessageType]struct{}{channels.AdminText: {}},
				map[channels.MessageType]struct{}{channels.Pinned: {}},
			},
			channels.NotifyAll: {
				map[channels.MessageType]struct{}{},
				map[channels.MessageType]struct{}{channels.AdminText: {}, channels.Pinned: {}},
			},
		},
	}

	for _, mt := range types {
		for _, level := range levels {
			chanID, _ := id.NewRandomID(rng, id.User)
			msgHash := make([]byte, 24)
			rng.Read(msgHash)
			asymIdentifier := append(chanID.Marshal(), []byte("AsymmToPublicBcast")...)
			symIdentifier := append(chanID.Marshal(), []byte("SymmetricBroadcast")...)
			pubKey, _, err := ed25519.GenerateKey(rng)
			if err != nil {
				t.Fatalf("Failed to generate Ed25519 keys: %+v", err)
			}
			tags := []string{fmt.Sprintf("%x-usrping", pubKey)}

			nuj.NotificationFilters = append(nuj.NotificationFilters,
				channels.NotificationFilter{
					Identifier: asymIdentifier,
					ChannelID:  chanID,
					Tags:       tags,
					AllowLists: notificationLevelAllowLists["asymmetric"][level],
				},
				channels.NotificationFilter{
					Identifier: symIdentifier,
					ChannelID:  chanID,
					Tags:       tags,
					AllowLists: notificationLevelAllowLists["symmetric"][level],
				})

			if _, exists := notificationLevelAllowLists["symmetric"][level].AllowWithTags[mt]; !exists {
				break
			}
			state := clientNotif.Mute
			if level != channels.NotifyNone {
				state = clientNotif.Push
			}

			nuj.ChangedNotificationStates = append(nuj.ChangedNotificationStates, channels.NotificationState{
				ChannelID: chanID,
				Level:     level,
				Status:    state,
			})
		}
	}

	nuj.DeletedNotificationStates = append(nuj.DeletedNotificationStates, id.NewIdFromString("deleted", id.User, t))
	nuj.MaxState = clientNotif.Push

	data, err := json.MarshalIndent(nuj, "//  ", "  ")
	if err != nil {
		t.Errorf("Failed to JSON %T: %+v", nuj, err)
	} else {
		fmt.Printf("//  %s\n", data)
	}
}

// Produces example JSON of AdminKeysUpdateJSON to be used for documentation.
func Test_BuildJSON_AdminKeysUpdateJSON(t *testing.T) {
	rng := rand.New(rand.NewSource(67915687))

	channelID, _ := id.NewRandomID(rng, id.User)

	jsonable := AdminKeysUpdateJSON{
		ChannelId: channelID,
		IsAdmin:   true,
	}

	data, err := json.MarshalIndent(jsonable, "//  ", "  ")
	if err != nil {
		t.Errorf("Failed to JSON %T: %+v", jsonable, err)
	} else {
		fmt.Printf("//  %s\n", data)
	}
}

// Produces example JSON of DmTokenUpdateJSON to be used for documentation.
func Test_BuildJSON_DmTokenUpdateJSON(t *testing.T) {
	rng := rand.New(rand.NewSource(67915687))

	channelID, _ := id.NewRandomID(rng, id.User)

	jsonable := DmTokenUpdateJSON{
		ChannelId: channelID,
		SendToken: true,
	}

	data, err := json.MarshalIndent(jsonable, "//  ", "  ")
	if err != nil {
		t.Errorf("Failed to JSON %T: %+v", jsonable, err)
	} else {
		fmt.Printf("//  %s\n", data)
	}
}

// Produces example JSON of ChannelUpdateJSON to be used for documentation.
func Test_BuildJSON_ChannelUpdateJSON(t *testing.T) {
	rng := rand.New(rand.NewSource(72928915))

	channelID, _ := id.NewRandomID(rng, id.User)

	jsonable := ChannelUpdateJSON{
		ChannelID: channelID,
		Deleted:   false,
	}

	data, err := json.MarshalIndent(jsonable, "//  ", "  ")
	if err != nil {
		t.Errorf("Failed to JSON %T: %+v", jsonable, err)
	} else {
		fmt.Printf("//  %s\n", data)
	}
}

// Produces example JSON of MessageReceivedJSON to be used for documentation.
func Test_BuildJSON_MessageReceivedJSON(t *testing.T) {
	rng := rand.New(rand.NewSource(72928915))

	channelID, _ := id.NewRandomID(rng, id.User)

	jsonable := MessageReceivedJSON{
		UUID:      rng.Int63(),
		ChannelID: channelID,
		Update:    false,
	}

	data, err := json.MarshalIndent(jsonable, "//  ", "  ")
	if err != nil {
		t.Errorf("Failed to JSON %T: %+v", jsonable, err)
	} else {
		fmt.Printf("//  %s\n", data)
	}
}

// Produces example JSON of UserMutedJSON to be used for documentation.
func Test_BuildJSON_UserMutedJSON(t *testing.T) {
	rng := rand.New(rand.NewSource(72928915))

	channelID, _ := id.NewRandomID(rng, id.User)
	pubkey, _, _ := ed25519.GenerateKey(rng)

	jsonable := UserMutedJSON{
		ChannelID: channelID,
		PubKey:    pubkey,
		Unmute:    true,
	}

	data, err := json.MarshalIndent(jsonable, "//  ", "  ")
	if err != nil {
		t.Errorf("Failed to JSON %T: %+v", jsonable, err)
	} else {
		fmt.Printf("//  %s\n", data)
	}
}

// Produces example JSON of MessageDeletedJSON to be used for documentation.
func Test_BuildJSON_MessageDeletedJSON(t *testing.T) {
	rng := rand.New(rand.NewSource(72928915))

	deletedID := message.ID{}
	rng.Read(deletedID[:])

	jsonable := MessageDeletedJSON{
		MessageID: deletedID,
	}

	data, err := json.MarshalIndent(jsonable, "//  ", "  ")
	if err != nil {
		t.Errorf("Failed to JSON %T: %+v", jsonable, err)
	} else {
		fmt.Printf("//  %s\n", data)
	}
}
