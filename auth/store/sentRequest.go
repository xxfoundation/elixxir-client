////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	sidhinterface "gitlab.com/elixxir/client/v5/interfaces/sidh"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentSentRequestVersion = 1

type SentRequest struct {
	kv *versioned.KV

	partner                 *id.ID
	partnerHistoricalPubKey *cyclic.Int
	myPrivKey               *cyclic.Int
	myPubKey                *cyclic.Int
	mySidHPrivKeyA          *sidh.PrivateKey
	mySidHPubKeyA           *sidh.PublicKey
	fingerprint             format.Fingerprint
	reset                   bool

	mux sync.Mutex
}

type sentRequestDisk struct {
	PartnerHistoricalPubKey []byte
	MyPrivKey               []byte
	MyPubKey                []byte
	MySidHPrivKeyA          []byte
	MySidHPubKeyA           []byte
	Fingerprint             []byte
	Reset                   bool
}

func newSentRequest(kv *versioned.KV, partner *id.ID, partnerHistoricalPubKey,
	myPrivKey, myPubKey *cyclic.Int, sidHPrivA *sidh.PrivateKey,
	sidHPubA *sidh.PublicKey, fp format.Fingerprint, reset bool) (*SentRequest, error) {

	sr := &SentRequest{
		kv:                      kv,
		partner:                 partner,
		partnerHistoricalPubKey: partnerHistoricalPubKey,
		myPrivKey:               myPrivKey,
		myPubKey:                myPubKey,
		mySidHPubKeyA:           sidHPubA,
		mySidHPrivKeyA:          sidHPrivA,
		fingerprint:             fp,
		reset:                   reset,
	}

	return sr, sr.save()
}

func loadSentRequest(kv *versioned.KV, partner *id.ID, grp *cyclic.Group) (*SentRequest, error) {

	srKey := makeSentRequestKey(partner)
	obj, err := kv.Get(srKey, currentSentRequestVersion)

	// V0 Upgrade Path
	if !kv.Exists(err) {
		upgradeErr := upgradeSentRequestKeyV0(kv, partner)
		if upgradeErr != nil {
			return nil, errors.Wrapf(err, "%+v", upgradeErr)
		}
		obj, err = kv.Get(srKey, currentSentRequestVersion)
	}

	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"SentRequest Auth with %s", partner)
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
	// jww.INFO.Printf("loadSentRequest myPrivKey: %s",
	// 	hex.EncodeToString(myPrivKey.Bytes()))
	jww.INFO.Printf("loadSentRequest myPubKey: %s",
		hex.EncodeToString(myPubKey.Bytes()))
	jww.INFO.Printf("loadSentRequest fingerprint: %s",
		hex.EncodeToString(fp[:]))

	return &SentRequest{
		kv:                      kv,
		partner:                 partner,
		partnerHistoricalPubKey: historicalPubKey,
		myPrivKey:               myPrivKey,
		myPubKey:                myPubKey,
		mySidHPrivKeyA:          mySidHPrivKeyA,
		mySidHPubKeyA:           mySidHPubKeyA,
		fingerprint:             fp,
		reset:                   srd.Reset,
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
	// jww.INFO.Printf("saveSentRequest myPrivKey: %s",
	// 	hex.EncodeToString(sr.myPrivKey.Bytes()))
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
		Reset:                   sr.reset,
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

	return sr.kv.Set(makeSentRequestKey(sr.partner), &obj)
}

func (sr *SentRequest) delete() {
	if err := sr.kv.Delete(makeSentRequestKey(sr.partner),
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

func (sr *SentRequest) IsReset() bool {
	return sr.reset
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

func (sr *SentRequest) getType() RequestType {
	return Sent
}

// makeSentRequestKey makes the key string for accessing the
// partners sent request object from the key value store.
func makeSentRequestKey(partner *id.ID) string {
	return "sentRequest:" + partner.String()
}

// V0 Utility Functions

// makeSentRequestKeyV0 The old key used the string pattern
// "Partner:PartnerID" instead of "sentRequest:PartnerID".
func makeSentRequestKeyV0(partner *id.ID) string {
	return fmt.Sprintf("Partner:%v", partner.String())
}

// upgradeSentRequestKeyV0 upgrads the srKey from version 0 to 1 by
// changing the version number.
func upgradeSentRequestKeyV0(kv *versioned.KV, partner *id.ID) error {
	oldKey := makeSentRequestKeyV0(partner)
	obj, err := kv.Get(oldKey, 0)
	if err != nil {
		return err
	}

	jww.INFO.Printf("Upgrading legacy srKey for %s", partner)

	// Note: uses same encoding, just different keys
	obj.Version = 1
	err = kv.Set(makeSentRequestKey(partner), obj)
	if err != nil {
		return err
	}

	return kv.Delete(oldKey, 0)
}
