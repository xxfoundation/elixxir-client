package utility

import (
	//"encoding/base64"
	"fmt"
	//"io"

	//jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"

	"gitlab.com/elixxir/client/ctidh"
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/client/storage/versioned"
)

const currentCTIDHVersion = 0

////
// Public Key Storage utility functions
////

const currentCTIDHPubKeyVersion = 0

var mynike nike.Nike = ctidh.NewCtidhNike()

// StoreCTIDHPublicKey stores the given public key in the kv.
func StoreCTIDHPublicKey(kv *versioned.KV, publicKey nike.PublicKey, key string) error {
	now := netTime.Now()

	obj := versioned.Object{
		Version:   currentCTIDHPubKeyVersion,
		Timestamp: now,
		Data:      publicKey.Bytes(),
	}

	return kv.Set(key, &obj)
}

// LoadCTIDHPubKeyA loads a public key from storage.
func LoadCTIDHPublicKey(kv *versioned.KV, key string) (nike.PublicKey, error) {
	vo, err := kv.Get(key, currentCTIDHPubKeyVersion)
	if err != nil {
		return nil, err
	}

	pubKey, err := mynike.UnmarshalBinaryPublicKey(vo.Data)
	if err != nil {
		return nil, err
	}

	return pubKey, nil
}

// DeleteCTIDHPubKey removes the key from the store
func DeleteCTIDHPublicKey(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentCTIDHPubKeyVersion)
}

func MakeCTIDHPublicKeyKey(cid *id.ID) string {
	return fmt.Sprintf("CTIDHPubKey:%s", cid)
}

////
// Private Key Storage utility functions
////

const currentCTIDHPrivKeyVersion = 0

// StoreCTIDHPrivateKey is a helper to store the requestor public key.
func StoreCTIDHPrivateKey(kv *versioned.KV, privateKey nike.PrivateKey, key string) error {
	now := netTime.Now()

	obj := versioned.Object{
		Version:   currentCTIDHPrivKeyVersion,
		Timestamp: now,
		Data:      privateKey.Bytes(),
	}

	return kv.Set(key, &obj)
}

// LoadCTIDHPrivateKeyA loads a public key from storage.
func LoadCTIDHPrivateKey(kv *versioned.KV, key string) (nike.PrivateKey, error) {
	vo, err := kv.Get(key, currentCTIDHPrivKeyVersion)
	if err != nil {
		return nil, err
	}

	privKey, err := mynike.UnmarshalBinaryPrivateKey(vo.Data)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// DeleteCTIDHPrivateKey removes the key from the store
func DeleteCTIDHPrivateKey(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentCTIDHPrivKeyVersion)
}

func MakeCTIDHPrivateKeyKey(cid *id.ID) string {
	return fmt.Sprintf("CTIDHPrivKey:%s", cid)
}
