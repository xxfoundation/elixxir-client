package auth

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const currentSentRequestVersion = 0

type SentRequest struct {
	kv *versioned.KV

	partner                 *id.ID
	partnerhistoricalPubKey *cyclic.Int
	myPrivKey               *cyclic.Int
	myPubKey                *cyclic.Int
	fingerprint             format.Fingerprint
	sentMux                 sync.Mutex
}

type sentRequestDisk struct {
	PartnerhistoricalPubKey []byte
	MyPrivKey               []byte
	MyPubKey                []byte
	Fingerprint             []byte
}

func loadSentRequest(kv *versioned.KV, partner *id.ID, grp *cyclic.Group) (*SentRequest, error) {
	obj, err := kv.Get(versioned.MakePartnerPrefix(partner))
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"SentRequest Auth with %s", partner)
	}

	srd := &sentRequestDisk{}

	if err := json.Unmarshal(obj.Data, srd); err != nil {
		return nil, errors.WithMessagef(err, "Failed to Unmarshal "+
			"SentRequest Auth with %s", partner)
	}

	historicalPrivKey := grp.NewInt(1)
	if err = historicalPrivKey.GobDecode(srd.PartnerhistoricalPubKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode historical"+
			" private key with %s for SentRequest Auth", partner)
	}

	myPrivKey := grp.NewInt(1)
	if err = myPrivKey.GobDecode(srd.MyPubKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode private "+
			"key with %s for SentRequest Auth", partner)
	}

	myPubKey := grp.NewInt(1)
	if err = myPubKey.GobDecode(srd.MyPubKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode public "+
			"key with %s for SentRequest Auth", partner)
	}

	fp := format.Fingerprint{}
	copy(fp[:], srd.Fingerprint)

	return &SentRequest{
		kv:                      kv,
		partner:                 partner,
		partnerhistoricalPubKey: historicalPrivKey,
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

	historicalPrivKey, err := sr.myPubKey.GobEncode()
	if err != nil {
		return err
	}

	ipd := sentRequestDisk{
		PartnerhistoricalPubKey: historicalPrivKey,
		MyPrivKey:               privKey,
		MyPubKey:                pubKey,
		Fingerprint:             sr.fingerprint[:],
	}

	data, err := json.Marshal(&ipd)
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentSentRequestVersion,
		Timestamp: time.Now(),
		Data:      data,
	}

	return sr.kv.Set(versioned.MakePartnerPrefix(sr.partner), &obj)
}

func (sr *SentRequest) delete() error {
	return sr.kv.Delete(versioned.MakePartnerPrefix(sr.partner))
}

func (sr *SentRequest) GetPartner() *id.ID {
	return sr.partner
}

func (sr *SentRequest) GetPartnerHistoricalPubKey() *cyclic.Int {
	return sr.partnerhistoricalPubKey
}

func (sr *SentRequest) GetMyPrivKey() *cyclic.Int {
	return sr.myPrivKey
}

func (sr *SentRequest) GetMyPubKey() *cyclic.Int {
	return sr.myPubKey
}

func (sr *SentRequest) GetFingerprint() format.Fingerprint {
	return sr.fingerprint
}
