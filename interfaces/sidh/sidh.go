package interfaces

import "github.com/cloudflare/circl/dh/sidh"

const KeyId = sidh.Fp503

var PubKeyByteSize = sidh.NewPublicKey(sidh.Fp503,
	sidh.KeyVariantSidhA).Size()
var PubKeyBitSize = PubKeyByteSize * 8
var PrivKeyByteSize = sidh.NewPrivateKey(sidh.Fp503,
	sidh.KeyVariantSidhA).Size()
var PrivKeyBitSize = PrivKeyByteSize * 8
