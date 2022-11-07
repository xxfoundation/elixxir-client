package channels

import (
	"crypto/ed25519"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/hash"
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
	privKey := m.me.Privkey
	toStore := hashPrivateKey(privKey)
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

// hashPrivateKey is a helper function which generates a DM token.
// As per spec, this is just a hash of the private key.
func hashPrivateKey(privKey *ed25519.PrivateKey) []byte {
	h, err := hash.NewCMixHash()
	if err != nil {
		jww.FATAL.Panicf("Failed to generate cMix hash: %+v", err)
	}

	h.Write(privKey.Seed())
	return h.Sum(nil)
}
