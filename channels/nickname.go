package channels

import (
	"encoding/json"
	"errors"
	"sync"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	nicknameStoreStorageKey     = "nicknameStoreStorageKey"
	nicknameStoreStorageVersion = 0
)

type nicknameManager struct {
	byChannel map[id.ID]string
	mux       sync.RWMutex
	kv        *versioned.KV
}

// LoadOrNewNicknameManager returns the stored nickname manager if there is one
// or returns a new one.
func LoadOrNewNicknameManager(kv *versioned.KV) *nicknameManager {
	nm := &nicknameManager{
		byChannel: make(map[id.ID]string),
		kv:        kv,
	}

	err := nm.load()
	if err != nil && nm.kv.Exists(err) {
		jww.FATAL.Panicf("[CH] Failed to load nicknameManager: %+v", err)
	}

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

	nm.byChannel[*channelID] = nickname
	return nm.save()
}

// DeleteNickname removes the nickname for a given channel. The name will revert
// back to the codename for this channel instead.
func (nm *nicknameManager) DeleteNickname(channelID *id.ID) error {
	nm.mux.Lock()
	defer nm.mux.Unlock()

	delete(nm.byChannel, *channelID)

	return nm.save()
}

// GetNickname returns the nickname for the given channel if it exists.
func (nm *nicknameManager) GetNickname(channelID *id.ID) (
	nickname string, exists bool) {
	nm.mux.RLock()
	defer nm.mux.RUnlock()

	nickname, exists = nm.byChannel[*channelID]
	return
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

	return nm.kv.Set(nicknameStoreStorageKey, obj)
}

// load restores the nickname manager from disk.
func (nm *nicknameManager) load() error {
	obj, err := nm.kv.Get(nicknameStoreStorageKey, nicknameStoreStorageVersion)
	if err != nil {
		return err
	}

	list := make([]channelIDToNickname, 0)
	err = json.Unmarshal(obj.Data, &list)
	if err != nil {
		return err
	}

	for i := range list {
		current := list[i]
		nm.byChannel[current.ChannelId] = current.Nickname
	}

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
