package channels

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const (
	nicknameStoreStorageKey     = "nicknameStoreStorageKey"
	nicknameStoreStorageVersion = 0

	nicknamePrefix = "nickname"

	nicknameMapName    = "nicknameMap"
	nicknameMapVersion = 0
)

type nicknameManager struct {
	byChannel map[id.ID]string
	mux       sync.RWMutex
	callback  func(channelId *id.ID, nickname string, exists bool)
	remote    versioned.KV
}

// loadOrNewNicknameManager returns the stored nickname manager if there is one
// or returns a new one.
func loadOrNewNicknameManager(remote versioned.KV, callback func(channelId *id.ID,
	nickname string, exists bool)) *nicknameManager {
	kvRemote, err := remote.Prefix(nicknamePrefix)
	if err != nil {
		jww.FATAL.Panicf("Nicknames failed to prefix KV (remote)")
	}

	nm := &nicknameManager{
		byChannel: make(map[id.ID]string),
		remote:    kvRemote,
		callback:  callback,
	}

	nm.mux.Lock()
	err = nm.remote.ListenOnRemoteMap(nicknameMapName, nicknameMapVersion,
		nm.mapUpdate)
	if err != nil && nm.remote.Exists(err) {
		jww.FATAL.Panicf("[CH] Failed to load and listen to remote "+
			"updates on nicknameManager: %+v", err)
	}

	nm.mux.Unlock()

	return nm
}

// SetNickname sets the nickname in a channel after checking that the nickname
// is valid using [IsNicknameValid].
func (nm *nicknameManager) SetNickname(nickname string, channelID *id.ID) error {
	nm.mux.Lock()
	defer nm.mux.Unlock()

	if err := IsNicknameValid(nickname); err != nil {
		return err
	}

	if err := nm.setNicknameUnsafe(nickname, channelID); err != nil {
		return err
	}

	go nm.callback(channelID, nickname, true)

	return nil
}

// DeleteNickname removes the nickname for a given channel. The name will revert
// back to the codename for this channel instead.
func (nm *nicknameManager) DeleteNickname(channelID *id.ID) error {
	nm.mux.Lock()
	defer nm.mux.Unlock()

	if err := nm.deleteNicknameUnsafe(channelID); err != nil {
		return err
	}

	go nm.callback(channelID, "", false)

	return nil
}

// GetNickname returns the nickname for the given channel if it exists.
func (nm *nicknameManager) GetNickname(channelID *id.ID) (
	nickname string, exists bool) {
	nm.mux.RLock()
	defer nm.mux.RUnlock()

	nickname, exists = nm.byChannel[*channelID]
	return
}

//////////////////////////////////////////////////////////////////////////////
// Internal Nickname Changes Tracker                                       //
/////////////////////////////////////////////////////////////////////////////

// nicknameUpdates is a tracker for any modified channel nickname. This
// is used by [nicknameManager.mapUpdate] and every element of [modified]
// is reported as a [nicknameUpdate] to the [UpdateNicknames] callback.
type nicknameUpdates struct {
	modified []nicknameUpdate
}

// newNicknameUpdates is a constructor for nicknameUpdates.
func newNicknameUpdates() *nicknameUpdates {
	return &nicknameUpdates{
		modified: make([]nicknameUpdate, 0),
	}
}

// AddDeletion creates a [nicknameUpdate] report for a deleted channel nickname.
func (nc *nicknameUpdates) AddDeletion(chanId *id.ID) {
	nc.modified = append(nc.modified, nicknameUpdate{
		ChannelId:      chanId,
		Nickname:       "",
		NicknameExists: false,
	})
}

// AddCreatedOrEdit creates a [nicknameUpdate] report for a new or modified
// channel nickname.
func (nc *nicknameUpdates) AddCreatedOrEdit(nickname string, chanId id.ID) {
	nc.modified = append(nc.modified, nicknameUpdate{
		ChannelId:      &chanId,
		Nickname:       nickname,
		NicknameExists: true,
	})
}

// mapUpdate handles map updates, handles by versioned.KV's ListenOnRemoteMap
// method.
func (nm *nicknameManager) mapUpdate(edits map[string]versioned.ElementEdit) {

	nm.mux.Lock()
	defer nm.mux.Unlock()

	updates := newNicknameUpdates()
	for elementName, edit := range edits {
		// unmarshal element name
		chanId, err := unmarshalChID(elementName)
		if err != nil {
			jww.WARN.Printf("Failed to unmarshal id in nickname "+
				"update %s on operation %s , skipping: %+v", elementName,
				edit.Operation, err)
		}

		if edit.Operation == versioned.Deleted {
			if _, exists := nm.byChannel[*chanId]; !exists {
				// if we don't have it locally, skip
				continue
			}

			updates.AddDeletion(chanId)
			delete(nm.byChannel, *chanId)
			continue
		}

		var newUpdate string
		if err := json.Unmarshal(edit.NewElement.Data, &newUpdate); err != nil {
			jww.WARN.Printf("Failed to unmarshal data in nickname "+
				"update %s, skipping: %+v", elementName, err)
			continue
		}

		if edit.Operation == versioned.Created || edit.Operation == versioned.Updated {
			updates.AddCreatedOrEdit(newUpdate, *chanId)
		} else {
			jww.WARN.Printf("Failed to handle nickname update %s, "+
				"bad operation: %s, skipping", elementName, edit.Operation)
			continue
		}

		nm.upsertNicknameUnsafeRAM(chanId, newUpdate)
	}

	// Initiate callback
	if nm.callback != nil {
		nm.initiateCallbacks(updates)
	}
}

// initiateCallbacks is a helper function which reports every nicknameUpdate to
// the UpdateNicknames callback. It is acceptable for this to not be done in
// batch as modifications can be assumed to be independent of one another and
// should not create synchronization issues.
func (nm *nicknameManager) initiateCallbacks(updates *nicknameUpdates) {
	for _, edited := range updates.modified {
		go nm.callback(edited.ChannelId, edited.Nickname, edited.NicknameExists)
	}
}

// upsertNicknameUnsafeRAM is a helper function which memoizes channel updates
// to in RAM memory.
func (nm *nicknameManager) upsertNicknameUnsafeRAM(cID *id.ID, nickname string) {
	nm.byChannel[*cID] = nickname
}

// deleteNicknameUnsafe will remote the nickname into the remote local and into the
// memoized map.
func (nm *nicknameManager) deleteNicknameUnsafe(channelID *id.ID) error {
	if err := nm.remote.Delete(
		marshalChID(channelID), nicknameStoreStorageVersion); err != nil {
		return err
	}
	delete(nm.byChannel, *channelID)
	return nil
}

// setNicknameUnsafe will save the nickname into the remote local and into the
// memoized map.
func (nm *nicknameManager) setNicknameUnsafe(
	nickname string, channelID *id.ID) error {
	nm.byChannel[*channelID] = nickname
	data, err := json.Marshal(&nickname)
	if err != nil {
		return err
	}

	elementName := marshalChID(channelID)

	err = nm.remote.StoreMapElement(nicknameMapName, elementName,
		&versioned.Object{
			Version:   nicknameStoreStorageVersion,
			Timestamp: netTime.Now(),
			Data:      data,
		}, nicknameMapVersion)
	if err != nil {
		return err
	}

	return nil
}

// nicknameUpdate is a structure which reports how the channel's nickname
// has been modified.
type nicknameUpdate struct {
	ChannelId      *id.ID
	Nickname       string
	NicknameExists bool
}

func (nu nicknameUpdate) Equals(nu2 nicknameUpdate) bool {
	if nu.NicknameExists != nu2.NicknameExists {
		return false
	}

	if nu.Nickname != nu.Nickname {
		return false
	}

	if !nu.ChannelId.Cmp(nu2.ChannelId) {
		return false
	}

	return true
}

// IsNicknameValid checks if a nickname is valid.
//
// Rules:
//   - A nickname must not be longer than 24 characters.
//   - A nickname must not be shorter than 1 character.
//
// TODO: Add character filtering.
func IsNicknameValid(nick string) error {
	runeNick := []rune(nick)
	if len(runeNick) > 24 {
		return errors.New("nicknames must be 24 characters in length or less")
	}

	if len(runeNick) < 1 {
		return errors.New("nicknames must be at least 1 character in length")
	}

	return nil
}

func marshalChID(chID *id.ID) string {
	idBytes := chID.Marshal()
	return base64.StdEncoding.EncodeToString(idBytes)
}

func unmarshalChID(s string) (*id.ID, error) {
	chID := &id.ID{}
	if _, err := base64.StdEncoding.Decode(chID[:], []byte(s)); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode id of nickname")
	}
	return chID, nil
}
