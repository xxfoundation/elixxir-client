package ratchet

import (
	"errors"
)

type xxratchet struct {
	// FIXME: we have needs to both lookup this way and
	//        lookup state of a specific session id.
	sendStates    map[NegotiationState][]ID
	invSendStates map[ID]NegotiationState

	sendRatchets map[ID]SendRatchet
	recvRatchets map[ID]ReceiveRatchet

	rekeyTrigger RekeyTrigger
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
