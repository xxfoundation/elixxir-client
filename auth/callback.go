package auth

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type RequestCallback func(requestor contact.Contact, message string)
type ConfirmCallback func(partner contact.Contact)

func RegisterCallbacks(rcb RequestCallback, ccb ConfirmCallback,
	sw interfaces.Switchboard, storage *storage.Session) stoppable.Stoppable {

	rawMessages := make(chan message.Receive, 1000)
	sw.RegisterChannel("Auth", &id.ID{}, message.Raw, rawMessages)

	stop := stoppable.NewSingle("Auth")

	go func() {
		select {
		case <-stop.Quit():
			return
		case msg := <-rawMessages:
			//check the message is well formed
			if msglen(msg.Payload) != 2*
				cmixMsg := format.Unmarshal(msg.Payload)
		}

	}()

}
