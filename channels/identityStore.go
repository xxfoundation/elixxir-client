package channels

import (
	"gitlab.com/elixxir/client/v4/collective/versioned"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	identityStoreStorageKey     = "identityStoreStorageKey"
	identityStoreStorageVersion = 0
)

func storeIdentity(kv versioned.KV, ident cryptoChannel.PrivateIdentity) error {
	data := ident.Marshal()
	obj := &versioned.Object{
		Version:   identityStoreStorageVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return kv.Set(identityStoreStorageKey, obj)
}

func loadIdentity(kv versioned.KV) (cryptoChannel.PrivateIdentity, error) {
	obj, err := kv.Get(identityStoreStorageKey, identityStoreStorageVersion)
	if err != nil {
		return cryptoChannel.PrivateIdentity{}, err
	}
	return cryptoChannel.UnmarshalPrivateIdentity(obj.Data)
}
