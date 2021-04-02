///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package e2e

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

func makeRelationshipFingerprint(t RelationshipType, grp *cyclic.Group,
	myPrivKey, partnerPubKey *cyclic.Int, me, partner *id.ID) []byte {

	myPubKey := grp.ExpG(myPrivKey, grp.NewIntFromUInt(1))

	switch t {
	case Send:
		return e2e.MakeRelationshipFingerprint(myPubKey, partnerPubKey,
			me, partner)
	case Receive:
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
		Version:   currentRelationshipFingerprintVersion,
		Timestamp: now,
		Data:      fp,
	}

	return kv.Set(relationshipFingerprintKey, currentRelationshipVersion,
		&obj)
}

func loadRelationshipFingerprint(kv *versioned.KV) []byte {
	obj, err := kv.Get(relationshipFingerprintKey,
		currentRelationshipVersion)
	if err != nil {
		jww.FATAL.Panicf("Failed to load relationshipFingerprint at %s: "+
			"%s", kv.GetFullKey(relationshipFingerprintKey,
			currentRelationshipFingerprintVersion), err)
	}
	return obj.Data
}
