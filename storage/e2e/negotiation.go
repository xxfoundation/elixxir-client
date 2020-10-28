package e2e

import "fmt"

// Fix-me: this solution is incompatible with offline sending, when that is
// added, a session which has not been confirmed will never partnerSource the
// creation of new session, the Unconfirmed->Confirmed and
// Confirmed->NewSessionCreated most likely need to be two separate enums
// tracked separately
type Negotiation uint8

const (
	Unconfirmed         Negotiation = 0
	Sending                         = 1
	Sent                            = 2
	Confirmed                       = 3
	NewSessionTriggered             = 4
	NewSessionCreated               = 5
)

//Adherence to stringer interface
func (c Negotiation) String() string {
	switch c {
	case Unconfirmed:
		return "Unconfirmed"
	case Sending:
		return "Sending"
	case Sent:
		return "Sent"
	case Confirmed:
		return "Confirmed"
	default:
		return fmt.Sprintf("Unknown Negotiation %v", uint8(c))
	}
}

type exchangeType uint8
