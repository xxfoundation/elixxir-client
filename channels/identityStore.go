package channels

import (
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	identityStoreStorageKey     = "identityStoreStorageKey"
	identityStoreStorageVersion = 0
)

func storeIdentity(kv *utility.KV, ident cryptoChannel.PrivateIdentity,
	storageTag string) error {
	data := ident.Marshal()
	obj := &versioned.Object{
		Version:   identityStoreStorageVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return kv.Set(makeIdentityKvKey(storageTag), obj.Marshal())
}

func loadIdentity(kv *utility.KV,
	storageTag string) (cryptoChannel.PrivateIdentity, error) {
	data, err := kv.Get(makeIdentityKvKey(storageTag),
		identityStoreStorageVersion)
	if err != nil {
		return cryptoChannel.PrivateIdentity{}, err
	}
	return cryptoChannel.UnmarshalPrivateIdentity(data)
}

func makeIdentityKvKey(tag string) string {
	return tag + identityStoreStorageKey
}
