package ud

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/utility"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/protobuf/runtime/protoimpl"
	"time"
	jww "github.com/spf13/jwalterweatherman"
)

type lookupCallback func(contact.Contact, error)


func (m *Manager)lookupProcess(c chan message.Receive, quitCh <-chan struct{}){
	for true {
		select {
		case <-quitCh:
			return
		case response := <-c:
			// edge check the encryption
			if response.Encryption!=message.E2E{
				jww.WARN.Printf("Dropped a lookup response from user " +
					"discovery due to incorrect encryption")
			}

			// unmarshal the message
			lookupResponse := &LookupResponse{}
			if err :=proto.Unmarshal(response.Payload, lookupResponse); err!=nil{
				jww.WARN.Printf("Dropped a lookup response from user " +
					"discovery due to failed unmarshal: %s", err)
			}

			// get the appropriate channel from the lookup
			m.inProgressMux.RLock()
			ch, ok := m.inProgressLookup[lookupResponse.CommID]
			m.inProgressMux.RUnlock()
			if !ok{
				jww.WARN.Printf("Dropped a lookup response from user " +
					"discovery due to unknown comm ID: %d",
					lookupResponse.CommID)
			}

			// send the response on the correct channel
			// drop if the send cannot be completed
			select{
				case ch<-lookupResponse:
				default:
					jww.WARN.Printf("Dropped a lookup response from user " +
						"discovery due failure to transmit to handling thread: " +
						"commID: %d", lookupResponse.CommID)
			}
		}
	}
}


// returns the public key of the passed id as known by the user discovery system
// or returns by the timeout
func (m *Manager)Lookup(id *id.ID, callback lookupCallback, timeout time.Duration)error{

	//get the id of this comm so it can be connected to its responce
	commID, err := m.getCommID()
	if err!=nil{
		return errors.WithMessage(err, "Random generation failed")
	}

	//build the request
	request := &LookupSend{
		UserID:        id.Marshal(),
		CommID:        commID,
	}

	requestMarshaled, err := proto.Marshal(request)
	if err!=nil{
		return errors.WithMessage(err, "Failed to form outgoing request")
	}

	msg := message.Send{
		Recipient:   m.udID,
		Payload:     requestMarshaled,
		MessageType: message.UdLookup,
	}

	//register the request in the responce map so it can be procesed on return
	responseChan := make(chan *LookupResponse, 1)
	m.inProgressMux.Lock()
	m.inProgressLookup[commID] = responseChan
	m.inProgressMux.Unlock()

	//send the request
	rounds, _, err := m.net.SendE2E(msg, params.GetDefaultE2E())

	if err!=nil{
		return errors.WithMessage(err, "Failed to send the lookup " +
			"request")
	}

	//register the round event to capture if the round fails
	roundFailChan := make(chan dataStructures.EventReturn, len(rounds))

	for _, round := range rounds{
		//subtract a millisecond to ensure this timeout will trigger before
		// the one below
		m.net.GetInstance().GetRoundEvents().AddRoundEventChan(round,
			roundFailChan, timeout-1*time.Millisecond, states.FAILED)
	}

	//start the go routine which will trigger the callback
	go func(){
		timer := time.NewTimer(timeout)

		var err error
		var c contact.Contact

		select{
			//return an error if the round fails
			case <-roundFailChan:
				err= errors.New("One or more rounds failed to " +
					"resolve, lookup not delivered")
			//return an error if the timeout is reached
			case <-timer.C:
				err= errors.New("Response from User Discovery" +
					" did not come before timeout")
			//return the contact if one is returned
			case response := <-responseChan:
				if response.Error!=""{
					err = errors.Errorf("User Discovery returned an " +
						"error on Lookup: %s", response.Error)
				}else{
					pubkey := m.grp.NewIntFromBytes(response.PubKey)
					c = contact.Contact{
						ID:             id,
						DhPubKey:       pubkey,
						OwnershipProof: nil,
						Facts:          nil,
					}
				}
		}
		//delete the response channel from the map
		m.inProgressMux.Lock()
		delete(m.inProgressLookup, commID)
		m.inProgressMux.Unlock()
		//call the callback last in case it is blocking
		callback(c, err)
	}()

	return nil
}