////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/interfaces/nike"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
)

type ReceivedRequest struct {
	kv *versioned.KV

	// contact of partner
	partner contact.Contact

	// PQ Public key of partner
	theirPQPubKey nike.PublicKey

	//round received on
	round rounds.Round

	//lock to make sure only one operator at a time
	mux sync.Mutex
}

func newReceivedRequest(kv *versioned.KV, c contact.Contact,
	key nike.PublicKey, round rounds.Round) *ReceivedRequest {

	if err := util.StoreContact(kv, c); err != nil {
		jww.FATAL.Panicf("Failed to save contact for partner %s: %+v", c.ID.String(), err)
	}

	pqStoreKey := util.MakePQPublicKeyKey(c.ID)
	if err := util.StorePQPublicKey(kv, key, pqStoreKey); err != nil {
		jww.FATAL.Panicf("Failed to save contact PQ pubKey for "+
			"partner %s: %+v", c.ID.String(), err)
	}

	roundStoreKey := makeRoundKey(c.ID)
	if err := rounds.StoreRound(kv, round, roundStoreKey); err != nil {
		jww.FATAL.Panicf("Failed to save round request was received on "+
			"for partner %s: %+v", c.ID.String(), err)
	}

	return &ReceivedRequest{
		kv:            kv,
		partner:       c,
		theirPQPubKey: key,
		round:         round,
	}
}

func loadReceivedRequest(kv *versioned.KV, partner *id.ID) (
	*ReceivedRequest, error) {

	c, err := util.LoadContact(kv, partner)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"Received Auth Request Contact with %s",
			partner)
	}

	key, err := util.LoadPQPublicKey(kv,
		util.MakePQPublicKeyKey(partner))
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"Received Auth Request Partner PQ key with %s",
			partner)
	}

	round, err := rounds.LoadRound(kv, makeRoundKey(partner))
	if err != nil && kv.Exists(err) {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"round request was received on with %s",
			partner)
	} else if err != nil && !kv.Exists(err) {
		jww.WARN.Printf("No round info for partner %s", partner)
	}

	return &ReceivedRequest{
		kv:            kv,
		partner:       c,
		theirPQPubKey: key,
		round:         round,
	}, nil
}

func (rr *ReceivedRequest) GetContact() contact.Contact {
	return rr.partner
}

func (rr *ReceivedRequest) GetTheirPQPubKey() nike.PublicKey {
	return rr.theirPQPubKey
}

func (rr *ReceivedRequest) GetRound() rounds.Round {
	return rr.round
}

func (rr *ReceivedRequest) delete() {
	if err := util.DeleteContact(rr.kv, rr.partner.ID); err != nil {
		jww.FATAL.Panicf("Failed to delete received request "+
			"contact for %s", rr.partner.ID)
	}
	if err := util.DeletePQPublicKey(rr.kv,
		util.MakePQPublicKeyKey(rr.partner.ID)); err != nil {
		jww.FATAL.Panicf("Failed to delete received request "+
			"SIDH pubkey for %s", rr.partner.ID)
	}
}

func (rr *ReceivedRequest) getType() RequestType {
	return Receive
}

func makeRoundKey(partner *id.ID) string {
	return "receivedRequestRound:" + partner.String()
}
