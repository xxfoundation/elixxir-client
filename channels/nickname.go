package channels

import (
	"encoding/json"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const (
	nicknameStoreStorageKey     = "nicknameStoreStorageKey"
	nicknameStoreStorageVersion = 0
)

type nicknameManager struct {
	byChannel map[id.ID]string

	mux sync.RWMutex

	kv *versioned.KV
}

// loadOrNewNicknameManager returns the stored nickname manager if there is
// one or returns a new one
func loadOrNewNicknameManager(kv *versioned.KV) *nicknameManager {
	nm := &nicknameManager{
		byChannel: make(map[id.ID]string),
		kv:        kv,
	}
	err := nm.load()
	if nm.kv.Exists(err) {
		jww.FATAL.Panicf("Failed to load nicknameManager: %+v", err)
	}

	return nm

}

// GetNickname returns the nickname for the given channel if it exists
func (nm *nicknameManager) GetNickname(ch *id.ID) (
	nickname string, exists bool) {
	nm.mux.RLock()
	defer nm.mux.RUnlock()

	nickname, exists = nm.byChannel[*ch]
	return
}

// SetNickname sets the nickname for a channel after checking that the nickname
// is valid using IsNicknameValid
func (nm *nicknameManager) SetNickname(newNick string, ch *id.ID) error {
	nm.mux.Lock()
	defer nm.mux.Unlock()

	if err := IsNicknameValid(newNick); err != nil {
		return err
	}

	nm.byChannel[*ch] = newNick
	return nm.save()
}

// DeleteNickname removes the nickname for a given channel, using the codename
// for that channel instead
func (nm *nicknameManager) DeleteNickname(ch *id.ID) error {
	nm.mux.Lock()
	defer nm.mux.Unlock()

	delete(nm.byChannel, *ch)

	return nm.save()
}

// channelIDToNickname is a serialization structure. This is used by the save
// and load functions to serialize the nicknameManager's byChannel map.
type channelIDToNickname struct {
	channelId id.ID
	nickname  string
}

// save stores the nickname manager to disk. The caller of this must
// hold the mux.
func (nm *nicknameManager) save() error {
	list := make([]channelIDToNickname, 0)
	for chId, nickname := range nm.byChannel {
		list = append(list, channelIDToNickname{
			channelId: chId,
			nickname:  nickname,
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
		nm.byChannel[current.channelId] = current.nickname
	}

	return nil
}

// IsNicknameValid checks if a nickname is valid
//
// rules
//   - a Nickname must not be longer than 24 characters
// todo: add character filtering
func IsNicknameValid(nm string) error {
	if len([]rune(nm)) > 24 {
		return errors.New("nicknames must be 24 characters in length or less")
	}

	return nil
}
