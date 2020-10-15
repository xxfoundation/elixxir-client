package auth

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const currentSentRequestVersion = 0

type SentRequest struct {
	kv *versioned.KV

	partner     *id.ID
	myPrivKey   *cyclic.Int
	myPubKey    *cyclic.Int
	fingerprint format.Fingerprint
}

type sentRequestDisk struct {
	MyPrivKey   []byte
	MyPubKey    []byte
	Fingerprint []byte
}

func loadSentRequest(kv *versioned.KV, partner *id.ID, grp *cyclic.Group) (*SentRequest, error) {
	obj, err := kv.Get(versioned.MakePartnerPrefix(partner))
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"SentRequest Auth with %s", partner)
	}

	ipd := &sentRequestDisk{}

	if err := json.Unmarshal(obj.Data, ipd); err != nil {
		return nil, errors.WithMessagef(err, "Failed to Unmarshal "+
			"SentRequest Auth with %s", partner)
	}

	myPrivKey := grp.NewInt(1)
	if err = myPrivKey.GobDecode(ipd.MyPubKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode private "+
			"key with %s for SentRequest Auth", partner)
	}

	myPubKey := grp.NewInt(1)
	if err = myPubKey.GobDecode(ipd.MyPubKey); err != nil {
		return nil, errors.WithMessagef(err, "Failed to decode public "+
			"key with %s for SentRequest Auth", partner)
	}

	fp := format.Fingerprint{}
	copy(fp[:], ipd.Fingerprint)

	return &SentRequest{
		kv:          kv,
		partner:     partner,
		myPrivKey:   myPrivKey,
		myPubKey:    myPubKey,
		fingerprint: fp,
	}, nil
}

func (ip *SentRequest) save() error {

	privKey, err := ip.myPrivKey.GobEncode()
	if err != nil {
		return err
	}

	pubKey, err := ip.myPubKey.GobEncode()
	if err != nil {
		return err
	}

	ipd := sentRequestDisk{
		MyPrivKey:   privKey,
		MyPubKey:    pubKey,
		Fingerprint: ip.fingerprint[:],
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

	return ip.kv.Set(versioned.MakePartnerPrefix(ip.partner), &obj)
}

func (ip *SentRequest) delete() error {
	return ip.kv.Delete(versioned.MakePartnerPrefix(ip.partner))
}

func (ip *SentRequest) GetPartner() *id.ID {
	return ip.partner
}

func (ip *SentRequest) GetMyPrivKey() *cyclic.Int {
	return ip.myPrivKey
}

func (ip *SentRequest) GetMyPubKey() *cyclic.Int {
	return ip.myPubKey
}

func (ip *SentRequest) GetFingerprint() format.Fingerprint {
	return ip.fingerprint
}
