package auth

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

type ReceivedRequest struct {
	kv *versioned.KV

	aid authIdentity

	// ID received on
	myID *id.ID

	// contact of partner
	partner contact.Contact

	//sidHPublic key of partner
	theirSidHPubKeyA *sidh.PublicKey
}

func loadReceivedRequest(kv *versioned.KV, partner *id.ID, myID *id.ID) (
	*ReceivedRequest, error) {

	// try the load with both the new prefix and the old, which one is
	// successful will determine which file structure the sent request will use
	// a change was made when auth was upgraded to handle auths for multiple
	// outgoing IDs and it became possible to have multiple auths for the same
	// partner at a time, so it now needed to be keyed on the touple of
	// partnerID,MyID. Old receivedByID always have the same myID so they can be left
	// at their own paths
	aid := makeAuthIdentity(partner, myID)
	newKV := kv
	oldKV := kv.Prefix(makeRequestPrefix(aid))

	c, err := util.LoadContact(newKV, partner)

	//loading with the new prefix path failed, try with the new
	if err != nil {
		c, err = util.LoadContact(newKV, partner)
		if err != nil {
			return nil, errors.WithMessagef(err, "Failed to Load "+
				"Received Auth Request Contact with %s and to %s",
				partner, myID)
		} else {
			kv = oldKV
		}
	} else {
		kv = newKV
	}

	key, err := util.LoadSIDHPublicKey(kv,
		util.MakeSIDHPublicKeyKey(c.ID))
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"Received Auth Request Partner SIDHkey with %s and to %s",
			partner, myID)
	}

	return &ReceivedRequest{
		aid:              aid,
		kv:               kv,
		myID:             myID,
		partner:          c,
		theirSidHPubKeyA: key,
	}, nil
}

func (rr *ReceivedRequest) getType() RequestType {
	return Receive
}
