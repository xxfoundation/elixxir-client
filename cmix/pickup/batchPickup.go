package pickup

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type pickupRequest struct {
	target *id.ID
	round  rounds.Round
	id     receptionID.EphemeralIdentity
}

func (m *pickup) processBatchMessagePickups(comms MessageRetrievalComms, stop *stoppable.Single) {
	maxBatchSize := 20
	batchDelay := 50 * time.Millisecond

	batch := make([]pickupRequest, 0, maxBatchSize)
	var timer = &time.Timer{
		C: make(<-chan time.Time),
	}

	for {
		shouldProcess := false

		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case <-timer.C:
			shouldProcess = true
		case req := <-m.gatewayMessageRequests:
			batch = append(batch, req)

			if len(batch) >= maxBatchSize {
				shouldProcess = true
			} else if len(batch) == 1 {
				timer = time.NewTimer(batchDelay)
			}
		}

		if !shouldProcess {
			continue
		}
		shouldProcess = false

		// Build batch message request
		msg := &pb.GetMessagesBatch{
			Requests: make([]*pb.GetMessages, len(batch)),
			Timeout:  500,
		}

		for i, v := range batch {
			msg.Requests[i] = &pb.GetMessages{
				ClientID: v.id.EphId[:],
				RoundID:  uint64(v.round.ID),
				Target:   v.target.Marshal(),
			}
		}

		// SEND & PROCESS
		resp, err := m.sender.SendToAny(func(host *connect.Host) (interface{}, error) {
			return comms.RequestBatchMessages(host, msg)
		}, stop)
		if err != nil {
			jww.ERROR.Printf("%+v", err)
			continue
		}

		batchResponse := resp.(*pb.GetMessagesResponseBatch)

		for i, result := range batchResponse.GetResults() {
			if result == nil {
				// TODO how to handle this
			}
			bundle, err := m.processPickupResponse(result, batch[i].id, batch[i].round)
			if err != nil {
				jww.ERROR.Printf("%+v", err)
				continue
			}

			if len(bundle.Messages) != 0 {
				rl := batch[i]
				// If successful and there are messages, we send them to another
				// thread
				bundle.Identity = receptionID.EphemeralIdentity{
					EphId:  rl.id.EphId,
					Source: rl.id.Source,
				}
				bundle.RoundInfo = rl.round
				m.messageBundles <- bundle

				jww.DEBUG.Printf("Removing round %d from unchecked store", rl.round.ID)
				err = m.unchecked.Remove(
					id.Round(rl.round.ID), rl.id.Source, rl.id.EphId)
				if err != nil {
					jww.ERROR.Printf("Could not remove round %d from "+
						"unchecked rounds store: %v", rl.round.ID, err)
				}
			}
		}

		batch = make([]pickupRequest, 0, maxBatchSize)
	}
}
