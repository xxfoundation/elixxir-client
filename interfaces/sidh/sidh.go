package interfaces

import "github.com/cloudflare/circl/dh/sidh"

const SidHKeyId = sidh.Fp503
var SidHPubKeyBitSize = sidh.NewPublicKey(sidh.Fp503, sidh.KeyVariantSidhA).Size()
var SidHPubKeyByteSize = SidHPubKeyBitSize /8
var SidHPrivKeyBitSize = sidh.NewPrivateKey(sidh.Fp503, sidh.KeyVariantSidhA).Size()
var SidHPrivKeyByteSize = SidHPrivKeyBitSize /8
