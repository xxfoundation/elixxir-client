///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/hex"
	"encoding/json"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	sidhinterface "gitlab.com/elixxir/client/interfaces/sidh"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const currentSentRequestVersion = 0

type SentRequest struct {
	kv *versioned.KV

	aid authIdentity

	myID                    *id.ID
	partner                 *id.ID
	partnerHistoricalPubKey *cyclic.Int
	myPrivKey               *cyclic.Int
	myPubKey                *cyclic.Int
	mySidHPrivKeyA          *sidh.PrivateKey
	mySidHPubKeyA           *sidh.PublicKey
	fingerprint             format.Fingerprint

	mux sync.Mutex
}

type sentRequestDisk struct {
	PartnerHistoricalPubKey []byte
	MyPrivKey               []byte
	MyPubKey                []byte
	MySidHPrivKeyA          []byte
	MySidHPubKeyA           []byte
	Fingerprint             []byte
}

func newSentRequest(kv *versioned.KV, partner, myID *id.ID, partnerHistoricalPubKey, myPrivKey,
	myPubKey *cyclic.Int, sidHPrivA *sidh.PrivateKey, sidHPubA *sidh.PublicKey,
	fp format.Fingerprint) (*SentRequest, error) {

	aid := makeAuthIdentity(partner, myID)

	sr := &SentRequest{
		kv:                      kv,
		aid:                     aid,
		partner:                 partner,
		partnerHistoricalPubKey: partnerHistoricalPubKey,
		myPrivKey:               myPrivKey,
		myPubKey:                myPubKey,
		mySidHPubKeyA:           sidHPubA,
		mySidHPrivKeyA:          sidHPrivA,
		fingerprint:             fp,
	}

	return sr, sr.save()
}

func loadSentRequest(kv *versioned.KV, partner *id.ID, myID *id.ID, grp *cyclic.Group) (*SentRequest, error) {

	// try the load with both the new prefix and the old, which one is
	// successful will determine which file structure the sent request will use
	// a change was made when auth was upgraded to handle auths for multiple
	// outgoing IDs and it became possible to have multiple auths for the same
	// partner at a time, so it now needed to be keyed on the touple of
	// partnerID,MyID. Old receivedByID always have the same myID so they can be left
	// at their own paths
	aid := makeAuthIdentity(partner, myID)

	obj, err := kv.Get(makeSentRequestKey(aid),
		currentSentRequestVersion)

	//loading with the new prefix path failed, try with the new
	if err != nil {
		obj, err = kv.Get(versioned.MakePartnerPrefix(partner),
			currentSentRequestVersion)
		if err != nil {
			return nil, errors.WithMessagef(err, "Failed to Load "+
				"SentRequest Auth with %s", partner)
		} else {
			err = kv.Set(makeSentRequestKey(aid), currentSentRequestVersion, obj)
			if err != nil {
				return nil, errors.WithMessagef(err, "Failed to update "+
					"from old store SentRequest Auth with %s", partner)
			}
		}
	}

	srd := &sentRequestDisk{}

	if err := json.Unmarshal(obj.Data, srd); err != nil {
		return nil, errors.WithMessagef(err, "Failed to Unmarshal "+
			"SentRequest Auth with %s", partner)
	}

	historicalPubKey := grp.NewInt(1)
	if err = historicalPubKey.GobDecode(srd.PartnerHistoricalPubKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode historical "+
			"private key with %s for SentRequest Auth", partner)
	}

	myPrivKey := grp.NewInt(1)
	if err = myPrivKey.GobDecode(srd.MyPrivKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode private key "+
			"with %s for SentRequest Auth", partner)
	}

	myPubKey := grp.NewInt(1)
	if err = myPubKey.GobDecode(srd.MyPubKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode public "+
			"key with %s for SentRequest Auth", partner)
	}

	mySidHPrivKeyA := sidh.NewPrivateKey(sidhinterface.KeyId,
		sidh.KeyVariantSidhA)
	if err = mySidHPrivKeyA.Import(srd.MySidHPrivKeyA); err != nil {
		return nil, errors.WithMessagef(err,
			"Failed to decode sidh private key "+
				"with %s for SentRequest Auth", partner)
	}

	mySidHPubKeyA := sidh.NewPublicKey(sidhinterface.KeyId,
		sidh.KeyVariantSidhA)
	if err = mySidHPubKeyA.Import(srd.MySidHPubKeyA); err != nil {
		return nil, errors.WithMessagef(err,
			"Failed to decode sidh public "+
				"key with %s for SentRequest Auth", partner)
	}

	fp := format.Fingerprint{}
	copy(fp[:], srd.Fingerprint)

	jww.INFO.Printf("loadSentRequest partner: %s",
		hex.EncodeToString(partner[:]))
	jww.INFO.Printf("loadSentRequest historicalPubKey: %s",
		hex.EncodeToString(historicalPubKey.Bytes()))
	jww.INFO.Printf("loadSentRequest myPrivKey: %s",
		hex.EncodeToString(myPrivKey.Bytes()))
	jww.INFO.Printf("loadSentRequest myPubKey: %s",
		hex.EncodeToString(myPubKey.Bytes()))
	jww.INFO.Printf("loadSentRequest fingerprint: %s",
		hex.EncodeToString(fp[:]))

	return &SentRequest{
		kv:                      kv,
		aid:                     aid,
		myID:                    myID,
		partner:                 partner,
		partnerHistoricalPubKey: historicalPubKey,
		myPrivKey:               myPrivKey,
		myPubKey:                myPubKey,
		mySidHPrivKeyA:          mySidHPrivKeyA,
		mySidHPubKeyA:           mySidHPubKeyA,
		fingerprint:             fp,
	}, nil
}

func (sr *SentRequest) save() error {
	privKey, err := sr.myPrivKey.GobEncode()
	if err != nil {
		return err
	}

	pubKey, err := sr.myPubKey.GobEncode()
	if err != nil {
		return err
	}

	historicalPubKey, err := sr.partnerHistoricalPubKey.GobEncode()
	if err != nil {
		return err
	}

	jww.INFO.Printf("saveSentRequest partner: %s",
		hex.EncodeToString(sr.partner[:]))
	jww.INFO.Printf("saveSentRequest historicalPubKey: %s",
		hex.EncodeToString(sr.partnerHistoricalPubKey.Bytes()))
	jww.INFO.Printf("saveSentRequest myPrivKey: %s",
		hex.EncodeToString(sr.myPrivKey.Bytes()))
	jww.INFO.Printf("saveSentRequest myPubKey: %s",
		hex.EncodeToString(sr.myPubKey.Bytes()))
	jww.INFO.Printf("saveSentRequest fingerprint: %s",
		hex.EncodeToString(sr.fingerprint[:]))

	sidHPriv := make([]byte, sidhinterface.PrivKeyByteSize)
	sidHPub := make([]byte, sidhinterface.PubKeyByteSize)
	sr.mySidHPrivKeyA.Export(sidHPriv)
	sr.mySidHPubKeyA.Export(sidHPub)

	ipd := sentRequestDisk{
		PartnerHistoricalPubKey: historicalPubKey,
		MyPrivKey:               privKey,
		MyPubKey:                pubKey,
		MySidHPrivKeyA:          sidHPriv,
		MySidHPubKeyA:           sidHPub,
		Fingerprint:             sr.fingerprint[:],
	}

	data, err := json.Marshal(&ipd)
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentSentRequestVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return sr.kv.Set(versioned.MakePartnerPrefix(sr.partner),
		currentSentRequestVersion, &obj)
}

func (sr *SentRequest) delete() {
	if err := sr.kv.Delete(versioned.MakePartnerPrefix(sr.partner),
		currentSentRequestVersion); err != nil {
		jww.FATAL.Panicf("Failed to delete sent request from %s to %s: "+
			"%+v", sr.partner, sr.partner, err)
	}
}

func (sr *SentRequest) GetPartner() *id.ID {
	return sr.partner
}

func (sr *SentRequest) GetPartnerHistoricalPubKey() *cyclic.Int {
	return sr.partnerHistoricalPubKey
}

func (sr *SentRequest) GetMyPrivKey() *cyclic.Int {
	return sr.myPrivKey
}

func (sr *SentRequest) GetMyPubKey() *cyclic.Int {
	return sr.myPubKey
}

func (sr *SentRequest) GetMySIDHPrivKey() *sidh.PrivateKey {
	return sr.mySidHPrivKeyA
}

func (sr *SentRequest) GetMySIDHPubKey() *sidh.PublicKey {
	return sr.mySidHPubKeyA
}

// OverwriteSIDHKeys is used to temporarily overwrite sidh keys
// to handle e.g., confirmation receivedByID.
// FIXME: this is a code smell but was the cleanest solution at
// the time. Business logic should probably handle this better?
func (sr *SentRequest) OverwriteSIDHKeys(priv *sidh.PrivateKey,
	pub *sidh.PublicKey) {
	sr.mySidHPrivKeyA = priv
	sr.mySidHPubKeyA = pub
}

func (sr *SentRequest) GetFingerprint() format.Fingerprint {
	return sr.fingerprint
}

func (sr *SentRequest) getAuthID() authIdentity {
	return sr.aid
}

func (sr *SentRequest) getType() RequestType {
	return Sent
}
