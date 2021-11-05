///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

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
)

const currentSentRequestVersion = 0

type SentRequest struct {
	kv *versioned.KV

	partner                 *id.ID
	partnerHistoricalPubKey *cyclic.Int
	myPrivKey               *cyclic.Int
	myPubKey                *cyclic.Int
	mySidHPrivKeyA          *sidh.PrivateKey
	mySidHPubKeyA           *sidh.PublicKey
	fingerprint             format.Fingerprint
}

type sentRequestDisk struct {
	PartnerHistoricalPubKey []byte
	MyPrivKey               []byte
	MyPubKey                []byte
	MySidHPrivKey			[]byte
	mySidHPubKey			[]byte
	Fingerprint             []byte
}

func loadSentRequest(kv *versioned.KV, partner *id.ID, grp *cyclic.Group) (*SentRequest, error) {
	obj, err := kv.Get(versioned.MakePartnerPrefix(partner),
		currentSentRequestVersion)
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

	mySidHPrivKeyA := sidh.NewPrivateKey(sidhinterface.SidHKeyId, sidh.KeyVariantSidhA)
	if err = mySidHPrivKeyA.Import(srd.MySidHPrivKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode sidh private key "+
			"with %s for SentRequest Auth", partner)
	}

	mySidHPubKeyA := sidh.NewPublicKey(sidhinterface.SidHKeyId, sidh.KeyVariantSidhA)
	if err = mySidHPubKeyA.Import(srd.mySidHPubKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode sidh public "+
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
		partner:                 partner,
		partnerHistoricalPubKey: historicalPubKey,
		myPrivKey:               myPrivKey,
		myPubKey:                myPubKey,
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

	sidHPriv := make([]byte, sidhinterface.SidHPrivKeyByteSize)
	sidHPub := make([]byte, sidhinterface.SidHPubKeyByteSize)
	sr.mySidHPrivKeyA.Export(sidHPriv)
	sr.mySidHPubKeyA.Export(sidHPub)

	ipd := sentRequestDisk{
		PartnerHistoricalPubKey: historicalPubKey,
		MyPrivKey:               privKey,
		MyPubKey:                pubKey,
		MySidHPrivKey: 		     sidHPriv,
		mySidHPubKey: 			 sidHPub,
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

func (sr *SentRequest) delete() error {
	return sr.kv.Delete(versioned.MakePartnerPrefix(sr.partner),
		currentSentRequestVersion)
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

func (sr *SentRequest) GetMySidhPrivKeyA() *sidh.PrivateKey {
	return sr.mySidHPrivKeyA
}

func (sr *SentRequest) GetMySidhPubKeyA() *sidh.PublicKey {
	return sr.mySidHPubKeyA
}

func (sr *SentRequest) GetFingerprint() format.Fingerprint {
	return sr.fingerprint
}
