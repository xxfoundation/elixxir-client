package ud

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/utility"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/protobuf/runtime/protoimpl"
	"time"
)

type lookupCallback func([]contact.Contact, error)

// returns the public key of the passed id as known by the user discovery system
// or returns by the timeout
func (m *Manager)Lookup(id *id.ID, callback lookupCallback, timeout time.Duration)error{

	commID, err := m.getCommID()
	if err!=nil{
		return errors.WithMessage(err, "Random generation failed")
	}


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

	rounds, mid, err := m.net.SendE2E(msg, params.GetDefaultE2E())

	if err!=nil{
		return errors.WithMessage(err, "Failed to send the lookup " +
			"request")
	}



	go func(){
		results :=
		utility.TrackResults()
	}



}