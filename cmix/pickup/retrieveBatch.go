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
	checkedGateways   []*id.ID
}

func (m *pickup) processBatchMessageRetrieval(comms MessageRetrievalComms, stop *stoppable.Single) {
	maxBatchSize := m.params.MaxBatchSize
	batchDelay := time.Duration(m.params.BatchDelay) * time.Millisecond

	batch := make([]*pickupRequest, 0, maxBatchSize)
	var timer = &time.Timer{
		C: make(<-chan time.Time),
	}

	for {
		shouldProcess := false

		var req *pickupRequest
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case <-timer.C:
			shouldProcess = true
		case rl := <-m.lookupRoundMessages: // Incoming lookup requests
			// Get shuffled list of gateways in round
			gwIds := m.getGatewayList(rl)

			// In testing, sometimes ignore requests to force retry
			if m.params.ForceMessagePickupRetry && m.shouldForceMessagePickupRetry(rl) {
				continue
			}

			// Initialize pickupRequest for new round
			req = &pickupRequest{gwIds[0], rl.Round, rl.Identity, gwIds[1:], nil}
		case req = <-m.gatewayMessageRequests: // Request retries
		}

		if req != nil {
			jww.DEBUG.Printf("[processBatchMessageRetrieval] Added round %d to batch", req.round.ID)

			// Add incoming request to batch
			batch = append(batch, req)
			if len(batch) >= maxBatchSize {
				shouldProcess = true
			} else if len(batch) == 1 {
				timer = time.NewTimer(batchDelay)
			}
		}

		// Continue unless batch is full or timer has elapsed
		if !shouldProcess {
			continue
		}

		jww.DEBUG.Printf("[processBatchMessageRetrieval] Checking for messages in rounds %+v", batch)

		// Reset timer & shouldProcess
		timer.Stop()
		timer = &time.Timer{
			C: make(<-chan time.Time),
		}
		shouldProcess = false

		// Build batch message request
		msg := &pb.GetMessagesBatch{
			Requests: make([]*pb.GetMessages, len(batch)),
			Timeout:  uint64(m.params.BatchPickupTimeout),
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
			jww.ERROR.Printf("Failed to request batch of messages: %+v", err)
			continue
		}

		// Process responses
		batchResponse := resp.(*pb.GetMessagesResponseBatch)
		for i, result := range batchResponse.GetResults() {
			proxiedRequest := batch[i]
			// Handler gw did not receive response in time/did not have contact with proxiedRequest
			if result == nil {
				jww.DEBUG.Printf("[processBatchMessageRetrieval] Handler gateway did not receive anything from target %s", proxiedRequest.target)
				go m.tryNextGateway(proxiedRequest)
				continue
			}

			// Handler gw encountered error getting messages from proxiedRequest
			respErr := batchResponse.GetErrors()[i]
			if respErr != "" {
				jww.ERROR.Printf("[processBatchMessageRetrieval] Handler gateway encountered error attempting to pick up messages from target %s: %s", proxiedRequest.target, respErr)
				go m.tryNextGateway(proxiedRequest)
				continue
			}

			// Process response from proxiedRequest gateway
			bundle, err := m.processPickupResponse(result, proxiedRequest.id, proxiedRequest.round.ID)
			if err != nil {
				jww.ERROR.Printf("[processBatchMessageRetrieval] Failed to process pickup response from proxiedRequest gateway %s: %+v", proxiedRequest.target, err)
				go m.tryNextGateway(proxiedRequest)
				continue
			}

			// Handle received bundle
			m.processBundle(bundle, proxiedRequest.id, proxiedRequest.round)
		}

		// Empty batch before restarting loop
		batch = make([]*pickupRequest, 0, maxBatchSize)
	}
}

// put pickup request back in batch, targeting next gateway in list of unchecked
func (m *pickup) tryNextGateway(req *pickupRequest) {
	if len(req.uncheckedGateways) == 0 {
		jww.ERROR.Printf("[processBatchMessageRetrieval] Failed to get pickup round %d "+
			"from all gateways (%v)", req.round.ID, append(req.checkedGateways, req.target))
		return
	}
	select {
	case m.gatewayMessageRequests <- &pickupRequest{
		target:            req.uncheckedGateways[0],
		round:             req.round,
		id:                req.id,
		uncheckedGateways: req.uncheckedGateways[1:],
		checkedGateways:   append(req.checkedGateways, req.target),
	}:
	default:
	}
}
