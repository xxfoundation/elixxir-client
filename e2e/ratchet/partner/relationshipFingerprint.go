////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package partner

import (
	jww "github.com/spf13/jwalterweatherman"
	session2 "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

func makeRelationshipFingerprint(t session2.RelationshipType, grp *cyclic.Group,
	myPrivKey, partnerPubKey *cyclic.Int, me, partner *id.ID) []byte {

	myPubKey := diffieHellman.GeneratePublicKey(myPrivKey, grp)

	switch t {
	case session2.Send:
		return e2e.MakeRelationshipFingerprint(myPubKey, partnerPubKey,
			me, partner)
	case session2.Receive:
		return e2e.MakeRelationshipFingerprint(myPubKey, partnerPubKey,
			partner, me)
	default:
		jww.FATAL.Panicf("Cannot built relationship fingerprint for "+
			"'%s'", t)
	}
	return nil
}

func storeRelationshipFingerprint(fp []byte, kv *versioned.KV) error {
	now := netTime.Now()
	obj := versioned.Object{
		Version:   currentRelationshipVersion,
		Timestamp: now,
		Data:      fp,
	}

	return kv.Set(relationshipFingerprintKey, &obj)
}

func loadRelationshipFingerprint(kv *versioned.KV) []byte {
	obj, err := kv.Get(relationshipFingerprintKey,
		currentRelationshipVersion)
	if err != nil {
		jww.FATAL.Panicf("cannot load relationshipFingerprint at %s: "+
			"%s", kv.GetFullKey(relationshipFingerprintKey,
			currentRelationshipFingerprintVersion), err)
	}
	return obj.Data
}

func deleteRelationshipFingerprint(kv *versioned.KV) error {
	return kv.Delete(relationshipFingerprintKey,
		currentRelationshipVersion)
}
