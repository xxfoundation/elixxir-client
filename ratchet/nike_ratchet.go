package ratchet

import (
	"errors"
	"fmt"

	"gitlab.com/xx_network/crypto/csprng"

	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/crypto/fastRNG"
)

type xxratchetFactory struct{}

var DefaultXXRatchetFactory = &xxratchetFactory{}

func NewXXRatchet(myPrivateKey nike.PrivateKey,
	myPublicKey nike.PublicKey, partnerPublicKey nike.PublicKey,
	params session.Params, rekeyTrigger RekeyTrigger,
	fpTracker FingerprintTracker) XXRatchet {
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	rng := rngGen.GetStream()

	size := uint32(params.MaxKeys)
	salt := make([]byte, 32)
	count, err := rng.Read(salt)
	if err != nil {
		panic(err)
	}
	if count != 32 {
		panic("rng failed")
	}

	r := newxxratchet(size, salt)

	threshold := uint32(float64(params.MaxKeys) * float64(params.RekeyThreshold))

	mySendRatchet := NewSendRatchet(myPrivateKey, myPublicKey, partnerPublicKey, salt, size, threshold)
	sendID := mySendRatchet.ID()

	myRecvRatchet := NewReceiveRatchet(myPrivateKey, partnerPublicKey, salt, size, threshold)
	recvID := myRecvRatchet.ID()

	for i := 0; i < int(NewSessionCreated); i++ {
		r.sendStates[NegotiationState(i)] = make([]ID, 0)
	}

	r.threshold = threshold
	r.sendStates[Unconfirmed] = []ID{sendID}
	r.invSendStates[sendID] = Unconfirmed
	r.sendRatchets[sendID] = mySendRatchet
	r.recvRatchets[recvID] = myRecvRatchet
	r.rekeyTrigger = rekeyTrigger
	r.fpTracker = fpTracker

	r.addKeyFingerprints(r.recvRatchets[recvID])

	return r
}

type xxratchet struct {
	size      uint32
	threshold uint32
	salt      []byte

	sendStates    map[NegotiationState][]ID
	invSendStates map[ID]NegotiationState

	sendRatchets map[ID]SendRatchet
	recvRatchets map[ID]ReceiveRatchet

	rekeyTrigger RekeyTrigger
	fpTracker    FingerprintTracker

	params session.Params
}

var _ XXRatchet = (*xxratchet)(nil)

func newxxratchet(size uint32, salt []byte) *xxratchet {
	return &xxratchet{
		size:          size,
		salt:          salt,
		sendStates:    make(map[NegotiationState][]ID),
		invSendStates: make(map[ID]NegotiationState),
		sendRatchets:  make(map[ID]SendRatchet),
		recvRatchets:  make(map[ID]ReceiveRatchet),
	}
}

func (x *xxratchet) Encrypt(id ID,
	plaintext []byte) (*EncryptedMessage, error) {
	message, err := x.sendRatchets[id].Encrypt(plaintext)
	switch err {
	case nil:
		return message, nil
	case ErrRekeyThreshold:
		myPubKey := x.rekeySender(id)
		x.rekeyTrigger.TriggerRekey(id, myPubKey)
		return message, nil
	default:
		return nil, err
	}
}

func (x *xxratchet) Decrypt(id ID,
	message *EncryptedMessage) (plaintext []byte, err error) {
	return x.recvRatchets[id].Decrypt(message)
}

func (x *xxratchet) rekeySender(id ID) nike.PublicKey {
	newSendRatchet := x.sendRatchets[id].Next()
	newid := newSendRatchet.ID()
	x.sendRatchets[newid] = newSendRatchet
	x.sendStates[Unconfirmed] = append(x.sendStates[Unconfirmed], newid)
	x.invSendStates[newid] = Unconfirmed
	return newSendRatchet.MyPublicKey()
}

// Rekey creates a new receiving ratchet defined
// by the received public key. This is called
// by case 6 above in our protocol description.
func (x *xxratchet) Rekey(oldReceiverRatchetID ID,
	theirPublicKey nike.PublicKey) (ID, error) {
	oldRatchet, ok := x.recvRatchets[oldReceiverRatchetID]
	if !ok {
		return ID{}, fmt.Errorf("receiving ratchet id not found: %s",
			oldReceiverRatchetID)
	}
	r := oldRatchet.Next(theirPublicKey)
	id := r.ID()
	x.recvRatchets[id] = r
	x.addKeyFingerprints(x.recvRatchets[id])
	return id, nil
}

func (x *xxratchet) SetState(senderRatchetID ID,
	newState NegotiationState) error {
	curState := x.invSendStates[senderRatchetID]
	if !curState.IsNewStateLegal(newState) {
		return errors.New("SetState: invalid state transition")
	}

	curList, ok := deleteRatchetIDFromList(senderRatchetID,
		x.sendStates[curState])
	if !ok {
		return errors.New("senderRatchetID not found")
	}
	x.sendStates[curState] = curList
	x.sendStates[newState] = append(x.sendStates[newState], senderRatchetID)
	x.invSendStates[senderRatchetID] = newState
	return nil
}

func (x *xxratchet) SendRatchets() []ID {
	ids := make([]ID, len(x.sendRatchets))
	i := 0
	for ratchetID, _ := range x.sendRatchets {
		ids[i] = ratchetID
		i++
	}
	return ids
}

func (x *xxratchet) SendRatchetsByState(state NegotiationState) []ID {
	return x.sendStates[state]
}

func (x *xxratchet) ReceiveRatchets() []ID {
	ids := make([]ID, len(x.recvRatchets))
	i := 0
	for ratchetID, _ := range x.recvRatchets {
		ids[i] = ratchetID
		i++
	}
	return ids
}

func (x *xxratchet) addKeyFingerprints(ratchet ReceiveRatchet) {
	fps := ratchet.DeriveFingerprints()
	for i := 0; i < len(fps); i++ {
		x.fpTracker.AddKey(fps[i])
	}
}

func deleteRatchetIDFromList(ratchetID ID,
	ratchets []ID) ([]ID, bool) {
	found := false
	for i := 0; i < len(ratchets); i++ {
		if ratchetID == ratchets[i] {
			head := ratchets[:i]
			tail := ratchets[i+1:]
			ratchets = append(head, tail...)
			found = true
			break
		}
	}
	return ratchets, found
}
