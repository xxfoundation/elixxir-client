////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"fmt"

	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	dmStoreKey     = "dmToken-%s"
	dmStoreVersion = 0
)

// enableDirectMessageToken is a helper functions for EnableDirectMessageToken
// which directly sets a token for the given channel ID into storage. This is an
// unsafe operation.
func (m *manager) enableDirectMessageToken(chId *id.ID) error {
	toStore := m.me.GetDMToken()
	vo := &versioned.Object{
		Version:   dmStoreVersion,
		Timestamp: netTime.Now(),
		Data:      toStore,
	}

	return m.kv.Set(createDmStoreKey(chId), vo)

}

// disableDirectMessageToken is a helper functions for DisableDirectMessageToken
// which deletes a token for the given channel ID into storage. This is an
// unsafe operation.
func (m *manager) disableDirectMessageToken(chId *id.ID) error {
	return m.kv.Delete(createDmStoreKey(chId), dmStoreVersion)
}

// getDmToken will retrieve a DM token from storage. If EnableDirectMessageToken
// has been called on this channel, then a token will exist in storage and be
// returned. If EnableDirectMessageToken has not been called on this channel,
// no token will exist and getDmToken will return nil.
func (m *manager) getDmToken(chId *id.ID) []byte {
	obj, err := m.kv.Get(createDmStoreKey(chId), dmStoreVersion)
	if err != nil {
		return nil
	}
	return obj.Data
}

func createDmStoreKey(chId *id.ID) string {
	return fmt.Sprintf(dmStoreKey, chId)

}
