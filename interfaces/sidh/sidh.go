package interfaces

import "github.com/cloudflare/circl/dh/sidh"

const SidHKeyId = sidh.Fp503
var SidHPubKeyByteSize = sidh.NewPublicKey(sidh.Fp503,
	sidh.KeyVariantSidhA).Size()
var SidHPubKeyBitSize = SidHPubKeyByteSize * 8
var SidHPrivKeyByteSize = sidh.NewPrivateKey(sidh.Fp503,
	sidh.KeyVariantSidhA).Size()
var SidHPrivKeyBitSize = SidHPrivKeyByteSize * 8
