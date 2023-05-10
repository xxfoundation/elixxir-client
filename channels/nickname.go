package channels

import (
	"encoding/json"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const (
	nicknameStoreStorageKey     = "nicknameStoreStorageKey"
	nicknameStoreStorageVersion = 0

	nicknameMapName    = "nicknameMap"
	nicknameMapVersion = 0
)

type nicknameManager struct {
	byChannel map[id.ID]string
	mux       sync.RWMutex
	callback  UpdateNicknames
	remote    versioned.KV
}

// LoadOrNewNicknameManager returns the stored nickname manager if there is one
// or returns a new one.
func LoadOrNewNicknameManager(kv versioned.KV) *nicknameManager {
	kvRemote, err := kv.Prefix(versioned.StandardRemoteSyncPrefix)
	if err != nil {
		jww.FATAL.Panicf("Nicknames failed to prefix KV (remote)")
	}

	nm := &nicknameManager{
		byChannel: make(map[id.ID]string),
		remote:    kvRemote,
	}

	nm.mux.Lock()
	loadedMap := nm.remote.ListenOnRemoteMap(nicknameMapName, nicknameMapVersion,
		nm.mapUpdate)
	err = nm.load(loadedMap)
	if err != nil && nm.remote.Exists(err) {
		jww.FATAL.Panicf("[CH] Failed to load nicknameManager: %+v", err)
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

	return nm.setNicknameUnsafe(nickname, channelID)
}

// DeleteNickname removes the nickname for a given channel. The name will revert
// back to the codename for this channel instead.
func (nm *nicknameManager) DeleteNickname(channelID *id.ID) error {
	nm.mux.Lock()
	defer nm.mux.Unlock()

	return nm.deleteNicknameUnsafe(channelID)
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

func newNicknameChanges() *nicknameUpdates {
	return &nicknameUpdates{
		modified: make([]NicknameUpdate, 0),
	}
}

type nicknameUpdates struct {
	modified []NicknameUpdate
}

func (nc *nicknameUpdates) AddDeletion(chanId *id.ID) {
	nc.modified = append(nc.modified, NicknameUpdate{
		ChannelId:      chanId,
		Nickname:       "",
		NicknameExists: false,
	})
}

func (nc *nicknameUpdates) AddCreatedOrEdit(nickname string, chanId id.ID) {
	nc.modified = append(nc.modified, NicknameUpdate{
		ChannelId:      &chanId,
		Nickname:       nickname,
		NicknameExists: true,
	})
}

func (nm *nicknameManager) mapUpdate(
	mapName string, edits map[string]versioned.ElementEdit) {

	// Ensure the user is attempting to modify the correct map
	if mapName != nicknameMapName {
		jww.ERROR.Printf("Got an update for the wrong map, "+
			"expected: %s, got: %s", nicknameMapName, mapName)
		return
	}

	nm.mux.Lock()
	defer nm.mux.Unlock()

	updates := newNicknameChanges()
	for elementName, edit := range edits {
		// unmarshal element name
		chanId := &id.ID{}
		if err := chanId.UnmarshalText([]byte(elementName)); err != nil {
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

		newUpdate := channelIDToNickname{}
		if err := json.Unmarshal(edit.NewElement.Data, &newUpdate); err != nil {
			jww.WARN.Printf("Failed to unmarshal data in nickname "+
				"update %s, skipping: %+v", elementName, err)
			continue
		}

		if edit.Operation == versioned.Created || edit.Operation == versioned.Updated {
			updates.AddCreatedOrEdit(newUpdate.Nickname, newUpdate.ChannelId)
		} else {
			jww.WARN.Printf("Failed to handle nickname update %s, "+
				"bad operation: %s, skipping", elementName, edit.Operation)
			continue
		}

		nm.upsertNicknameUnsafeRAM(newUpdate)
	}

	// Initiate callback
	if nm.callback != nil {
		nm.initiateCallbacks(updates)
	}
}

func (nm *nicknameManager) initiateCallbacks(updates *nicknameUpdates) {
	for _, edited := range updates.modified {
		go nm.callback(edited)
	}
}

func (nm *nicknameManager) upsertNicknameUnsafeRAM(newUpdate channelIDToNickname) {
	nm.byChannel[newUpdate.ChannelId] = newUpdate.Nickname
}

// channelIDToNickname is a serialization structure. This is used by the save
// and load functions to serialize the nicknameManager's byChannel map.
type channelIDToNickname struct {
	ChannelId id.ID
	Nickname  string
}

// save stores the nickname manager to disk. The caller of this must
// hold the mux.
func (nm *nicknameManager) save() error {
	list := make([]channelIDToNickname, 0)
	for channelID, nickname := range nm.byChannel {
		list = append(list, channelIDToNickname{
			ChannelId: channelID,
			Nickname:  nickname,
		})
	}

	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	obj := &versioned.Object{
		Version:   nicknameStoreStorageVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return nm.remote.Set(nicknameStoreStorageKey, obj)
}

// load restores the nickname manager from disk.
func (nm *nicknameManager) load(loadedMap map[string]*versioned.Object) error {

	for key, obj := range loadedMap {
		data := channelIDToNickname{}
		if err := json.Unmarshal(obj.Data, &data); err != nil {
			jww.WARN.Printf("Failed to unmarshal nickname "+
				"for %s, skipping: %+v", key, err)
		}

		nm.upsertNicknameUnsafeRAM(data)
	}

	return nil
}

func (nm *nicknameManager) deleteNicknameUnsafe(channelID *id.ID) error {
	if err := nm.remote.Delete(
		channelID.String(), nicknameStoreStorageVersion); err != nil {
		return err
	}
	delete(nm.byChannel, *channelID)
	return nil
}

func (nm *nicknameManager) setNicknameUnsafe(nickname string, channelID *id.ID) error {
	nm.byChannel[*channelID] = nickname
	data, err := json.Marshal(&channelIDToNickname{
		ChannelId: *channelID,
		Nickname:  nickname,
	})
	if err != nil {
		return err
	}

	err = nm.remote.Set(channelID.String(), &versioned.Object{
		Version:   nicknameStoreStorageVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	})
	if err != nil {
		return err
	}

	nm.byChannel[*channelID] = nickname
	return nil
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
