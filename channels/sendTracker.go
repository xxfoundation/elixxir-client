package channels

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/rounds"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

type trackedRound struct {
	msgID     cryptoChannel.MessageID
	channelID *id.ID
}

type trackedMessage struct {
	roundID   id.Round
	channelID *id.ID
}

type sendTracker struct {
	byRound map[id.Round][]trackedRound

	byMessageID map[cryptoChannel.MessageID]trackedMessage

	mux sync.Mutex

	em EventModel

	myUsername string

	net Client
}

func (sr *sendTracker) Send(channelID *id.ID,
	messageID cryptoChannel.MessageID, myUsername string,
	text string, timestamp time.Time, lease time.Duration,
	round rounds.Round) {

}

func (sr *sendTracker) handleSend(channelID *id.ID,
	messageID cryptoChannel.MessageID, round rounds.Round) {
	sr.mux.Lock()
	defer sr.mux.Unlock()

	//skip if already added
	_, existsMessage := sr.byMessageID[messageID]
	if existsMessage {
		return
	}

	//add the roundID
	roundsList, existsRound := sr.byRound[round.ID]
	sr.byRound[round.ID] = append(roundsList, trackedRound{messageID,
		channelID})

	//add the round
	sr.byMessageID[messageID] = trackedMessage{round.ID,
		channelID}

	if !existsRound {
		callback := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]cmix.RoundResult) {

		}
		sr.net.GetRoundResults(60 * time.Second)
	}

}

type roundResults struct {
	round id.Round
	st    *sendTracker
}

func (rr roundResults) callback(allRoundsSucceeded, timedOut bool, rounds map[id.Round]cmix.RoundResult) {
	rr.st.mux.Lock()

	//if the message was already handled, do nothing
	registered, existsRound := rr.st.byRound[rr.round]
	if !existsRound {
		rr.st.mux.Unlock()
		return
	}

	delete(rr.st.byRound, rr.round)

	for i := range registered {
		delete(rr.st.byMessageID, registered[i].msgID)
	}

	rr.st.mux.Unlock()

	for i := range registered {
		rr.st.eventModel.

	}

}

//deleteUnsafe deletes a tracked message from the database
func (sr *sendTracker) deleteUnsafe(channelID *id.ID,
	messageID cryptoChannel.MessageID, round rounds.Round) {
	sr.mux.Lock()
	defer sr.mux.Unlock()

	//skip if already added
	_, existsMessage := sr.byMessageID[messageID]
	if existsMessage {
		return
	}

	//add the roundID
	roundsList, existsRound := sr.byRound[round.ID]
	sr.byRound[round.ID] = append(roundsList, trackedRound{messageID,
		channelID})

	//add the round
	sr.byMessageID[messageID] = trackedMessage{round.ID,
		channelID}

	if !existsRound {
		callback := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]cmix.RoundResult) {
			sr.mux.Lock()
			defer sr.mux.Unlock()

			if !allRoundsSucceeded
		}
		sr.net.GetRoundResults(60 * time.Second)
	}

}
