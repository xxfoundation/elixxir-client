package ud

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type lookupCallback func(contact.Contact, error)

func (m *Manager) lookupProcess(c chan message.Receive, quitCh <-chan struct{}) {
	for true {
		select {
		case <-quitCh:
			return
		case response := <-c:

			// Unmarshal the message
			lookupResponse := &LookupResponse{}
			if err := proto.Unmarshal(response.Payload, lookupResponse); err != nil {
				jww.WARN.Printf("Dropped a lookup response from user "+
					"discovery due to failed unmarshal: %s", err)
			}

			// Get the appropriate channel from the lookup
			m.inProgressLookupMux.RLock()
			ch, ok := m.inProgressLookup[lookupResponse.CommID]
			m.inProgressLookupMux.RUnlock()
			if !ok {
				jww.WARN.Printf("Dropped a lookup response from user "+
					"discovery due to unknown comm ID: %d",
					lookupResponse.CommID)
			}

			// Send the response on the correct channel
			// Drop if the send cannot be completed
			select {
			case ch <- lookupResponse:
			default:
				jww.WARN.Printf("Dropped a lookup response from user "+
					"discovery due to failure to transmit to handling thread: "+
					"commID: %d", lookupResponse.CommID)
			}
		}
	}
}

// Lookup returns the public key of the passed ID as known by the user discovery
// system or returns by the timeout.
func (m *Manager) Lookup(uid *id.ID, callback lookupCallback, timeout time.Duration) error {

	if !m.IsRegistered(){
		return errors.New("Failed to lookup: " +
			"client is not registered")
	}

	// Get the ID of this comm so it can be connected to its response
	commID := m.getCommID()

	// Build the request
	request := &LookupSend{
		UserID: uid.Marshal(),
		CommID: commID,
	}

	requestMarshaled, err := proto.Marshal(request)
	if err != nil {
		return errors.WithMessage(err, "Failed to form outgoing request")
	}

	msg := message.Send{
		Recipient:   m.udID,
		Payload:     requestMarshaled,
		MessageType: message.UdLookup,
	}

	// Register the request in the response map so it can be processed on return
	responseChan := make(chan *LookupResponse, 1)
	m.inProgressLookupMux.Lock()
	m.inProgressLookup[commID] = responseChan
	m.inProgressLookupMux.Unlock()

	// Send the request
	rounds, err := m.net.SendUnsafe(msg, params.GetDefaultUnsafe())
	if err != nil {
		return errors.WithMessage(err, "Failed to send the lookup request")
	}

	// Register the round event to capture if the round fails
	roundFailChan := make(chan dataStructures.EventReturn, len(rounds))

	for _, round := range rounds {
		// Subtract a millisecond to ensure this timeout will trigger before the
		// one below
		m.net.GetInstance().GetRoundEvents().AddRoundEventChan(round,
			roundFailChan, timeout-1*time.Millisecond, states.FAILED,
			states.COMPLETED)
	}

	// Start the go routine which will trigger the callback
	go func() {
		timer := time.NewTimer(timeout)

		var err error
		var c contact.Contact

		done := false
		for !done {
			select {
			// Return an error if the round fails
			case fail := <-roundFailChan:
				if states.Round(fail.RoundInfo.State)==states.FAILED || fail.TimedOut{
					fType := ""
					if fail.TimedOut{
						fType = "timeout"
					}else{
						fType = fmt.Sprintf("round failure: %v", fail.RoundInfo.ID)
					}
					err = errors.Errorf("One or more rounds (%v) failed to " +
						"resolve due to: %s; search not delivered", rounds, fType)
					done = true
				}

			// Return an error if the timeout is reached
			case <-timer.C:
				err = errors.New("Response from User Discovery did not come " +
					"before timeout")
				done = true

			// Return the contact if one is returned
			case response := <-responseChan:
				if response.Error != "" {
					err = errors.Errorf("User Discovery returned an error on "+
						"lookup: %s", response.Error)
				} else {
					pubkey := m.grp.NewIntFromBytes(response.PubKey)
					c = contact.Contact{
						ID:             uid,
						DhPubKey:       pubkey,
						OwnershipProof: nil,
						Facts:          nil,
					}
				}
				done = true
			}
		}

		// Delete the response channel from the map
		m.inProgressLookupMux.Lock()
		delete(m.inProgressLookup, commID)
		m.inProgressLookupMux.Unlock()

		// Call the callback last in case it is blocking
		callback(c, err)
	}()

	return nil
}
