////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"encoding/base64"
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	jww "github.com/spf13/jwalterweatherman"
	sidhinterface "gitlab.com/elixxir/client/v4/interfaces/sidh"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
)

const currentSIDHVersion = 0

// NewSIDHPUblicKey is a helper which returns a proper new SIDH public key
// Right now this is set to Fp434 but it could change.
func NewSIDHPublicKey(variant sidh.KeyVariant) *sidh.PublicKey {
	return sidh.NewPublicKey(sidhinterface.KeyId, variant)
}

// NewSIDHPUblicKey is a helper which returns a proper new SIDH public key
// Right now this is set to Fp434 but it could change.
func NewSIDHPrivateKey(variant sidh.KeyVariant) *sidh.PrivateKey {
	return sidh.NewPrivateKey(sidhinterface.KeyId, variant)
}

// GetSIDHVariant returns the variant opposite the otherVariant
func GetCompatibleSIDHVariant(otherVariant sidh.KeyVariant) sidh.KeyVariant {
	// Note -- this is taken from inside the sidh lib to look for the A flag
	if (otherVariant & sidh.KeyVariantSidhA) == sidh.KeyVariantSidhA {
		return sidh.KeyVariantSidhB
	}
	return sidh.KeyVariantSidhA
}

// GenerateSIDHKeyPair generates a SIDH keypair
func GenerateSIDHKeyPair(variant sidh.KeyVariant, rng io.Reader) (
	*sidh.PrivateKey, *sidh.PublicKey) {
	priv := NewSIDHPrivateKey(variant)
	pub := NewSIDHPublicKey(variant)

	if err := priv.Generate(rng); err != nil {
		jww.FATAL.Panicf("Unable to generate SIDH private key: %+v",
			err)
	}
	priv.GeneratePublicKey(pub)
	return priv, pub
}

// String interface impl to dump the contents of the public key as b64 string
func StringSIDHPubKey(k *sidh.PublicKey) string {
	kBytes := make([]byte, k.Size())
	k.Export(kBytes)
	return base64.StdEncoding.EncodeToString(kBytes)
}

// String interface to dump the contents of the public key as b64 string
// NOTE: public key, not the private. We don't ever want to drop a
// private key into a log somewhere.
func StringSIDHPrivKey(k *sidh.PrivateKey) string {
	pubK := NewSIDHPublicKey(k.Variant())
	k.GeneratePublicKey(pubK)
	return StringSIDHPubKey(pubK)
}

////
// Public Key Storage utility functions
////

const currentSIDHPubKeyVersion = 0

// StoreSIDHPubKeyA is a helper to store the requestor public key (which is
// always of type A)
func StoreSIDHPublicKey(kv *versioned.KV, sidH *sidh.PublicKey, key string) error {
	now := netTime.Now()

	sidHBytes := make([]byte, sidH.Size()+1)
	sidHBytes[0] = byte(sidH.Variant())
	sidH.Export(sidHBytes[1:])

	obj := versioned.Object{
		Version:   currentSIDHPubKeyVersion,
		Timestamp: now,
		Data:      sidHBytes,
	}

	return kv.Set(key, &obj)
}

// LoadSIDHPubKeyA loads a public key from storage.
func LoadSIDHPublicKey(kv *versioned.KV, key string) (*sidh.PublicKey, error) {
	vo, err := kv.Get(key, currentSIDHPubKeyVersion)
	if err != nil {
		return nil, err
	}

	variant := sidh.KeyVariant(vo.Data[0])
	sidHPubkey := NewSIDHPublicKey(variant)
	return sidHPubkey, sidHPubkey.Import(vo.Data[1:])
}

// DeleteSIDHPubKey removes the key from the store
func DeleteSIDHPublicKey(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentSIDHPubKeyVersion)
}

func MakeSIDHPublicKeyKey(cid *id.ID) string {
	return fmt.Sprintf("SIDHPubKey:%s", cid)
}

////
// Private Key Storage utility functions
////

const currentSIDHPrivKeyVersion = 0

// StoreSIDHPrivateKeyA is a helper to store the requestor public key (which is
// always of type A)
func StoreSIDHPrivateKey(kv *versioned.KV, sidH *sidh.PrivateKey, key string) error {
	now := netTime.Now()

	sidHBytes := make([]byte, sidH.Size()+1)
	sidHBytes[0] = byte(sidH.Variant())
	sidH.Export(sidHBytes[1:])

	obj := versioned.Object{
		Version:   currentSIDHPrivKeyVersion,
		Timestamp: now,
		Data:      sidHBytes,
	}

	return kv.Set(key, &obj)
}

// LoadSIDHPrivateKeyA loads a public key from storage.
func LoadSIDHPrivateKey(kv *versioned.KV, key string) (*sidh.PrivateKey, error) {
	vo, err := kv.Get(key, currentSIDHPrivKeyVersion)
	if err != nil {
		return nil, err
	}

	variant := sidh.KeyVariant(vo.Data[0])
	sidHPrivkey := NewSIDHPrivateKey(variant)
	return sidHPrivkey, sidHPrivkey.Import(vo.Data[1:])
}

// DeleteSIDHPrivateKey removes the key from the store
func DeleteSIDHPrivateKey(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentSIDHPrivKeyVersion)
}

func MakeSIDHPrivateKeyKey(cid *id.ID) string {
	return fmt.Sprintf("SIDHPrivKey:%s", cid)
}
