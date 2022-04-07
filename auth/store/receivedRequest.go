package store

import (
	"encoding/base64"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/historical"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type ReceivedRequest struct {
	kv *versioned.KV

	// contact of partner
	partner contact.Contact

	//sidHPublic key of partner
	theirSidHPubKeyA *sidh.PublicKey

	//round received on
	round historical.Round

	//lock to make sure only one operator at a time
	mux *sync.Mutex
}

func newReceivedRequest(kv *versioned.KV, c contact.Contact,
	key *sidh.PublicKey, round historical.Round) *ReceivedRequest {

	if err := util.StoreContact(kv, c); err != nil {
		jww.FATAL.Panicf("Failed to save contact for partner %s", c.ID.String())
	}

	sidhStoreKey := util.MakeSIDHPublicKeyKey(c.ID)
	if err := util.StoreSIDHPublicKey(kv, key, sidhStoreKey); err != nil {
		jww.FATAL.Panicf("Failed to save contact SIDH pubKey for "+
			"partner %s", c.ID.String())
	}

	roundStoreKey := makeRoundKey(c.ID)
	if err := util.StoreRound(kv, round, roundStoreKey); err != nil {
		jww.FATAL.Panicf("Failed to save round request was received on "+
			"for partner %s", c.ID.String())
	}

	return &ReceivedRequest{
		kv:               kv,
		partner:          c,
		theirSidHPubKeyA: key,
		round:            round,
	}
}

func loadReceivedRequest(kv *versioned.KV, partner *id.ID) (
	*ReceivedRequest, error) {

	c, err := util.LoadContact(kv, partner)

	//loading with the new prefix path failed, try with the new
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"Received Auth Request Contact with %s",
			partner)
	}

	key, err := util.LoadSIDHPublicKey(kv, util.MakeSIDHPublicKeyKey(partner))
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"Received Auth Request Partner SIDHkey with %s",
			partner)
	}

	round, err := util.LoadRound(kv, makeRoundKey(partner))
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"round request was received on with %s",
			partner)
	}

	return &ReceivedRequest{
		kv:               kv,
		partner:          c,
		theirSidHPubKeyA: key,
		round:            round,
	}, nil
}

func (rr *ReceivedRequest) GetContact() contact.Contact {
	return rr.partner
}

func (rr *ReceivedRequest) GetTheirSidHPubKeyA() *sidh.PublicKey {
	return rr.theirSidHPubKeyA
}

func (rr *ReceivedRequest) GetRound() historical.Round {
	return rr.round
}

func (rr *ReceivedRequest) delete() {
	if err := util.DeleteContact(rr.kv, rr.partner.ID); err != nil {
		jww.FATAL.Panicf("Failed to delete received request "+
			"contact for %s", rr.partner.ID)
	}
	if err := util.DeleteSIDHPublicKey(rr.kv,
		util.MakeSIDHPublicKeyKey(rr.partner.ID)); err != nil {
		jww.FATAL.Panicf("Failed to delete received request "+
			"SIDH pubkey for %s", rr.partner.ID)
	}
}

func (rr *ReceivedRequest) getType() RequestType {
	return Receive
}

func (rr *ReceivedRequest) isTemporary() bool {
	return rr.kv.IsMemStore()
}

func makeRoundKey(partner *id.ID) string {
	return "receivedRequestRound:" +
		base64.StdEncoding.EncodeToString(partner.Marshal())
}
