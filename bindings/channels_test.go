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

func Test_buildJsons(t *testing.T) {
	rng := rand.New(rand.NewSource(43))

	// nickname update
	nickID := &id.ID{}
	rng.Read(nickID[:])

	nicknameJsonable := NickNameUpdateJson{
		ChannelId: nickID,
		Nickname:  "billNyeTheScienceGuy",
		Exists:    true,
	}

	nicknameJson, err := json.Marshal(&nicknameJsonable)
	if err != nil {
		t.Errorf("Failed to json nickname: %+v", err)
	} else {
		t.Logf("Nickname Json: %s", string(nicknameJson))
	}

	// Notifications update
	types := []channels.MessageType{channels.Text, channels.Pinned}
	levels := []channels.NotificationLevel{channels.NotifyPing, channels.NotifyAll}
	nuj := NotificationUpdateJson{
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
			for _, includeTags := range []bool{true, false} {
				for _, includeChannel := range []bool{true, false} {
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

					if includeChannel {
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

						if includeTags {
							if _, exists := notificationLevelAllowLists["symmetric"][level].AllowWithTags[mt]; !exists {
								break
							}
						} else if _, exists := notificationLevelAllowLists["symmetric"][level].AllowWithoutTags[mt]; !exists {
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
			}
		}
	}

	nuj.DeletedNotificationStates = append(nuj.DeletedNotificationStates, id.NewIdFromString("deleted", id.User, t))
	nuj.MaxState = clientNotif.Push

	notificationsJson, err := json.MarshalIndent(nuj, "", "\t")
	if err != nil {
		t.Errorf("Failed to json Notifications: %+v", err)
	} else {
		t.Logf("Notifications Json: %s", string(notificationsJson))
	}

	// MessageReceivedJson
	msgRcvdID := &id.ID{}
	rng.Read(msgRcvdID[:])

	messageRecieved := MessageReceivedJson{
		Uuid:      32,
		ChannelID: msgRcvdID,
		Update:    false,
	}
	messageRecievedJson, err := json.Marshal(&messageRecieved)
	if err != nil {
		t.Errorf("Failed to json message received: %+v", err)
	} else {
		t.Logf("MessageReceived Json: %s", string(messageRecievedJson))
	}

	// User Muted
	mutedID := &id.ID{}
	rng.Read(mutedID[:])

	pubkey, _, err := ed25519.GenerateKey(rng)
	if err != nil {
		t.Errorf("Failed to generate ed25519pubkey: %+v", err)
	}

	userMuted := UserMutedJson{
		ChannelID: mutedID,
		PubKey:    pubkey,
		Unmute:    false,
	}

	userMutedJson, err := json.Marshal(&userMuted)
	if err != nil {
		t.Errorf("Failed to json user muted: %+v", err)
	} else {
		t.Logf("UserMuted Json: %s", string(userMutedJson))
	}

	// message deleted
	deletedID := message.ID{}
	rng.Read(deletedID[:])

	msgDeleted := MessageDeletedJson{MessageID: deletedID}
	msgDeletedJson, err := json.Marshal(&msgDeleted)
	if err != nil {
		t.Errorf("Failed to json message deleted: %+v", err)
	} else {
		t.Logf("MessageDeleted Json: %s", string(msgDeletedJson))
	}

	channelUpdates := make([]ChannelsUpdateJson, 0, 5)
	channelUpdates = append(channelUpdates, ChannelsUpdateJson{
		ChannelId:        id.NewIdFromUInt(1, id.User, t),
		Status:           SyncCreated,
		BroadcastDMToken: false,
	})
	channelUpdates = append(channelUpdates, ChannelsUpdateJson{
		ChannelId:        id.NewIdFromUInt(2, id.User, t),
		Status:           SyncCreated,
		BroadcastDMToken: true,
	})
	channelUpdates = append(channelUpdates, ChannelsUpdateJson{
		ChannelId:        id.NewIdFromUInt(3, id.User, t),
		Status:           SyncUpdated,
		BroadcastDMToken: true,
	})
	channelUpdates = append(channelUpdates, ChannelsUpdateJson{
		ChannelId:        id.NewIdFromUInt(4, id.User, t),
		Status:           SyncUpdated,
		BroadcastDMToken: false,
	})
	channelUpdates = append(channelUpdates, ChannelsUpdateJson{
		ChannelId:        id.NewIdFromUInt(5, id.User, t),
		Status:           SyncDeleted,
		BroadcastDMToken: false,
	})

	channelUpdatesJson, err := json.Marshal(&channelUpdates)
	if err != nil {
		t.Errorf("Failed to json channel update: %+v", err)
	} else {
		t.Logf("[]ChannelUpdate Json: %s", string(channelUpdatesJson))
	}

}
