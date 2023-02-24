package pickup

import (
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type pickupRequest struct {
	target *id.ID
	round  id.Round
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

		msg := &pb.GetMessagesBatch{
			Requests: make([]*pb.GetMessages, len(batch)),
			Timeout:  500,
		}

		for i, v := range batch {
			msg.Requests[i] = &pb.GetMessages{
				ClientID: v.id.EphId[:],
				RoundID:  uint64(v.round),
				Target:   v.target.Marshal(),
			}
		}

		// SEND & PROCESS
		resp, err := m.sender.SendToAny(func(host *connect.Host) (interface{}, error) {
			return comms.RequestBatchMessages(host, msg)
		}, stop)
		if err != nil {

		}
		batchResponse := resp.(*pb.GetMessagesResponseBatch)

		batch = make([]pickupRequest, 0, maxBatchSize)
	}
}
