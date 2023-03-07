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

// processBatchMessageRetrieval is an alternative to processMessageRetrieval.
// It receives a roundLookup request and adds it to a batch, then pings a
// random gateway with the batch when either batch size == maxBatchSize or
// batchDelay milliseconds elapses after the first request is added to
// the batch.  If a pickup fails, it is returned to the batch targeting the
// next gateway in the batch.
func (m *pickup) processBatchMessageRetrieval(comms MessageRetrievalComms, stop *stoppable.Single) {
	maxBatchSize := m.params.MaxBatchSize
	batchDelay := time.Duration(m.params.BatchDelay) * time.Millisecond
	batch := make(map[id.Round]*pickupRequest)

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
		case rl := <-m.lookupRoundMessages:
			// Add incoming lookup reqeust to unchecked
			ri := rl.Round
			err := m.unchecked.AddRound(id.Round(ri.ID), ri.Raw,
				rl.Identity.Source, rl.Identity.EphId)
			if err != nil {
				jww.FATAL.Panicf(
					"Failed to denote Unchecked Round for round %d",
					id.Round(ri.ID))
			}

			// Get shuffled list of gateways in round
			gwIds := m.getGatewayList(rl)

			if m.params.ForceMessagePickupRetry && m.shouldForceMessagePickupRetry(rl) {
				// Do not add to the batch, leaving the round to be picked up in
				// unchecked round scheduler process
				continue
			}

			// Initialize pickupRequest for new round
			req = &pickupRequest{gwIds[0], rl.Round, rl.Identity, gwIds[1:], nil}
		case req = <-m.gatewayMessageRequests: // Request retries
		}

		if req != nil {
			rid := req.round.ID
			// Add incoming request to batch
			_, ok := batch[rid]
			if !ok {
				jww.DEBUG.Printf("[processBatchMessageRetrieval] Added round %d to batch", rid)
				batch[req.round.ID] = req
				if len(batch) >= maxBatchSize {
					shouldProcess = true
				} else if len(batch) == 1 {
					timer = time.NewTimer(batchDelay)
				}
			} else {
				jww.DEBUG.Printf("[processBatchMessageRetrieval] Ignoring request to add round %d; already in batch", rid)
			}

		}

		// Continue unless batch is full or timer has elapsed
		if !shouldProcess {
			continue
		}

		jww.TRACE.Printf("[processBatchMessageRetrieval] Sending batch message request for %d rounds", len(batch))

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

		orderedBatch := make([]*pickupRequest, len(batch))
		index := 0
		for _, v := range batch {
			orderedBatch[index] = v
			msg.Requests[index] = &pb.GetMessages{
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
			proxiedRequest := orderedBatch[i]
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
			bundle, err := m.buildMessageBundle(result, proxiedRequest.id, proxiedRequest.round.ID)
			if err != nil {
				jww.ERROR.Printf("[processBatchMessageRetrieval] Failed to process pickup response from proxiedRequest gateway %s: %+v", proxiedRequest.target, err)
				go m.tryNextGateway(proxiedRequest)
				continue
			}

			// Handle received bundle
			m.processBundle(bundle, proxiedRequest.id, proxiedRequest.round)
		}

		// Empty batch before restarting loop
		batch = make(map[id.Round]*pickupRequest)
	}
}

// tryNextGateway sends a pickupRequest back in the batch, targeting the next
// gateway in list of unchecked gateways.
func (m *pickup) tryNextGateway(req *pickupRequest) {
	// If there are no more unchecked gateways, log an error & return
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
