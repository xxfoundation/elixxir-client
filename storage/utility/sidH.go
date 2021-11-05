package utility

import (
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	sidhinterface "gitlab.com/elixxir/client/interfaces/sidh"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentSidHPubKeyAVersion = 0

func StoreSidHPubKeyA(kv *versioned.KV, id *id.ID, sidH *sidh.PublicKey) error {
	now := netTime.Now()

	sidHBytes := make([]byte, sidH.Size())
	sidH.Export(sidHBytes)

	obj := versioned.Object{
		Version:   currentSidHPubKeyAVersion,
		Timestamp: now,
		Data:      sidHBytes,
	}

	return kv.Set(makeSidHtKeyA(id), currentSidHPubKeyAVersion, &obj)
}

func LoadSidHPubKeyA(kv *versioned.KV, cid *id.ID) (*sidh.PublicKey, error) {
	vo, err := kv.Get(makeSidHtKeyA(cid), currentSidHPubKeyAVersion)
	if err != nil {
		return nil, err
	}

	sidHPubkey := sidh.NewPublicKey(sidhinterface.SidHKeyId, sidh.KeyVariantSidhA)
	return sidHPubkey, sidHPubkey.Import(vo.Data)
}

func DeleteSidHPubKeyA(kv *versioned.KV, cid *id.ID) error {
	return kv.Delete(makeSidHtKeyA(cid), currentSidHPubKeyAVersion)
}

func makeSidHtKeyA(cid *id.ID) string {
	return fmt.Sprintf("SidKPubKeyA:%s", cid)
}
