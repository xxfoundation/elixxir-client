package utility

import (
	"encoding/base64"
	"fmt"
	//"io"

	//jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"

	"gitlab.com/elixxir/client/ctidh"
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/client/storage/versioned"
)

const currentPQVersion = 0

////
// Public Key Storage utility functions
////

const currentPQPubKeyVersion = 0

var mynike nike.Nike = ctidh.NewCtidhNike()

// StorePQPublicKey stores the given public key in the kv.
func StorePQPublicKey(kv *versioned.KV, publicKey nike.PublicKey, key string) error {
	now := netTime.Now()

	obj := versioned.Object{
		Version:   currentPQPubKeyVersion,
		Timestamp: now,
		Data:      publicKey.Bytes(),
	}

	return kv.Set(key, &obj)
}

// LoadPQPubKeyA loads a public key from storage.
func LoadPQPublicKey(kv *versioned.KV, key string) (nike.PublicKey, error) {
	vo, err := kv.Get(key, currentPQPubKeyVersion)
	if err != nil {
		return nil, err
	}

	pubKey, err := mynike.UnmarshalBinaryPublicKey(vo.Data)
	if err != nil {
		return nil, err
	}

	return pubKey, nil
}

// DeletePQPubKey removes the key from the store
func DeletePQPublicKey(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentPQPubKeyVersion)
}

func MakePQPublicKeyKey(cid *id.ID) string {
	return fmt.Sprintf("PQPubKey:%s", cid)
}

////
// Private Key Storage utility functions
////

const currentPQPrivKeyVersion = 0

// StorePQPrivateKey is a helper to store the requestor public key.
func StorePQPrivateKey(kv *versioned.KV, privateKey nike.PrivateKey, key string) error {
	now := netTime.Now()

	obj := versioned.Object{
		Version:   currentPQPrivKeyVersion,
		Timestamp: now,
		Data:      privateKey.Bytes(),
	}

	return kv.Set(key, &obj)
}

// LoadPQPrivateKeyA loads a public key from storage.
func LoadPQPrivateKey(kv *versioned.KV, key string) (nike.PrivateKey, error) {
	vo, err := kv.Get(key, currentPQPrivKeyVersion)
	if err != nil {
		return nil, err
	}

	privKey, err := mynike.UnmarshalBinaryPrivateKey(vo.Data)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// DeletePQPrivateKey removes the key from the store
func DeletePQPrivateKey(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentPQPrivKeyVersion)
}

func MakePQPrivateKeyKey(cid *id.ID) string {
	return fmt.Sprintf("PQPrivKey:%s", cid)
}

// String interface impl to dump the contents of the public key as b64 string
func StringPQPubKey(k nike.PublicKey) string {
	return base64.StdEncoding.EncodeToString(k.Bytes())
}
