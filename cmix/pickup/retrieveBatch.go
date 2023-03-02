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
	target            *id.ID
	round             rounds.Round
	id                receptionID.EphemeralIdentity
	uncheckedGateways []*id.ID
}

func (m *pickup) processBatchMessageRetrieval(comms MessageRetrievalComms, stop *stoppable.Single) {
	maxBatchSize := 20
	batchDelay := 50 * time.Millisecond

	batch := make([]pickupRequest, 0, maxBatchSize)
	var timer = &time.Timer{
		C: make(<-chan time.Time),
	}

	for {
		shouldProcess := false

		var req pickupRequest
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case <-timer.C:
			shouldProcess = true
		case rl := <-m.lookupRoundMessages: // Incoming lookup requests
			gwIds := m.getGatewayList(rl)
			if m.params.ForceMessagePickupRetry && m.shouldForceMessagePickupRetry(rl) {
				continue
			}

			req = pickupRequest{gwIds[0], rl.Round, rl.Identity, gwIds[1:]}
		case req = <-m.gatewayMessageRequests: // Request retries
		}

		batch = append(batch, req)

		if len(batch) >= maxBatchSize {
			shouldProcess = true
		} else if len(batch) == 1 {
			timer = time.NewTimer(batchDelay)
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

		// Send batch pickup request to any gateway
		resp, err := m.sender.SendToAny(func(host *connect.Host) (interface{}, error) {
			return comms.RequestBatchMessages(host, msg)
		}, stop)
		if err != nil {
			jww.ERROR.Printf("%+v", err)
			continue
		}

		// Process responses
		batchResponse := resp.(*pb.GetMessagesResponseBatch)
		for i, result := range batchResponse.GetResults() {
			if result == nil {
				// TODO how to handle this (try next gw vs retry this target)
			}
			respErr := batchResponse.GetErrors()[i]
			if respErr != "" {
				go m.tryNextGateway(batch[i])
				continue
			}
			bundle, err := m.processPickupResponse(result, batch[i].id, batch[i].round)
			if err != nil {
				jww.ERROR.Printf("%+v", err)
				continue
			}

			m.processBundle(bundle, batch[i].id, batch[i].round)
		}

		batch = make([]pickupRequest, 0, maxBatchSize)
	}
}

func (m *pickup) tryNextGateway(req pickupRequest) {
	if len(req.uncheckedGateways) == 0 {
		// EXIT
		return
	}
	select {
	case m.gatewayMessageRequests <- pickupRequest{
		target:            req.uncheckedGateways[0],
		round:             req.round,
		id:                req.id,
		uncheckedGateways: req.uncheckedGateways[1:],
	}:
	default:
	}
}
