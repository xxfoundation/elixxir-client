////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/json"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	dmStoreKey     = "ChannelDMTokens"
	dmStoreVersion = 0
)

// saveDMTokens to the data store.
func (m *manager) saveDMTokens() error {
	toStore, err := json.Marshal(m.dmTokens)
	if err != nil {
		return err
	}
	vo := &versioned.Object{
		Version:   dmStoreVersion,
		Timestamp: netTime.Now(),
		Data:      toStore,
	}

	return m.kv.Set(dmStoreKey, vo)
}

// loadDMTokens from storage, or create a new dmTokens object.
func (m *manager) loadDMTokens() {
	obj, err := m.kv.Get(dmStoreKey, dmStoreVersion)
	if err != nil {
		jww.INFO.Printf("loading new dmTokens for channels: %v", err)
		m.dmTokens = make(map[id.ID]uint32)
		return
	}
	err = json.Unmarshal(obj.Data, &m.dmTokens)
	if err != nil {
		jww.ERROR.Printf("unmarshal channel dmTokens: %v", err)
		m.dmTokens = make(map[id.ID]uint32)
	}
}

// enableDirectMessageToken is a helper functions for EnableDirectMessageToken
// which directly sets a token for the given channel ID into storage. This is an
// unsafe operation.
func (m *manager) enableDirectMessageToken(chId *id.ID) error {
	token := m.me.GetDMToken()
	m.dmTokens[*chId] = token
	return m.saveDMTokens()
}

// disableDirectMessageToken is a helper functions for DisableDirectMessageToken
// which deletes a token for the given channel ID into storage. This is an
// unsafe operation.
func (m *manager) disableDirectMessageToken(chId *id.ID) error {
	delete(m.dmTokens, *chId)
	return m.saveDMTokens()
}

// getDmToken will retrieve a DM token from storage. If EnableDirectMessageToken
// has been called on this channel, then a token will exist in storage and be
// returned. If EnableDirectMessageToken has not been called on this channel,
// no token will exist and getDmToken will return nil.
func (m *manager) getDmToken(chId *id.ID) uint32 {
	token, ok := m.dmTokens[*chId]
	if !ok {
		return 0
	}
	return token
}
