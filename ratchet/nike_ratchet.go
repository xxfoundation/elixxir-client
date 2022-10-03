package ratchet

import (
	"errors"

	"gitlab.com/elixxir/client/interfaces/nike"
)

type xxratchet struct {
	size uint32
	salt []byte

	// FIXME: we have needs to both lookup this way and
	//        lookup state of a specific session id.
	sendStates    map[NegotiationState][]ID
	invSendStates map[ID]NegotiationState

	sendRatchets map[ID]SendRatchet
	recvRatchets map[ID]ReceiveRatchet

	rekeyTrigger RekeyTrigger
}

var _ XXRatchet = (*xxratchet)(nil)

func (x *xxratchet) Encrypt(id ID,
	plaintext []byte) (*EncryptedMessage, error) {
	return x.sendRatchets[id].Encrypt(plaintext)
}

func (x *xxratchet) Decrypt(id ID,
	message *EncryptedMessage) (plaintext []byte, err error) {
	return x.recvRatchets[id].Decrypt(message)
}

// Rekey creates a new receiving ratchet defined
// by the received public key. This is called
// by case 6 above in our protocol description.
//
// TODO: This function needs to send the new fingerprints
// to a callback interface method.
func (x *xxratchet) Rekey(oldReceiverRatchetID ID,
	theirPublicKey nike.PublicKey) (ID, nike.PublicKey) {

	myPrivateKey, myPublicKey := DefaultNIKE.NewKeypair()

	r, id := NewReceiveRatchet(myPrivateKey, theirPublicKey, x.salt, x.size)

	x.recvRatchets[id] = r

	return id, myPublicKey
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
	return nil
}

func (x *xxratchet) SendRatchets() []ID {
	ids := make([]ID, len(x.sendRatchets))
	i := 0
	for id, _ := range x.sendRatchets {
		ids[i] = id
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
	for id, _ := range x.recvRatchets {
		ids[i] = id
		i++
	}
	return ids
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
