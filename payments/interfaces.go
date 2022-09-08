package payments

import (
	"crypto/rsa"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

// Payments is the public interface of the payments package, used for
// sending & approving payment requests.
type Payments interface {
	Request(amount, address, description, xxnBlockchainDestination string,
		recipient *id.ID) (PaymentID, error)
	Approve(signedTransaction, data []byte, paymentId PaymentID,
		xxnetworkBlockchainDestination *id.ID) error
	GetRequest(id PaymentID) (PaymentRequest, bool)
	GetRequests() []PaymentRequest
}

// PaymentRequest is a public interface for stored payment data, allowing
// access to general info on the payment and its status
type PaymentRequest interface {
	GetInfo() *PaymentInfo
	GetStatus() Status
}

// PaymentID is a random identifier assigned to a payment request on creation
type PaymentID [32]byte

// Status type represents the status of a payment request
type Status uint8

const (
	Sent Status = iota
	Received
	Pending
	Submitted
	Accepted
	Rejected
)

func (s Status) String() string {
	switch s {
	case Sent:
		return "Sent"
	case Received:
		return "Received"
	case Pending:
		return "Pending"
	case Submitted:
		return "Submitted"
	case Accepted:
		return "Accepted"
	case Rejected:
		return "Rejected"
	default:
		return "Unknown"
	}
}

// ReceivePaymentCallback is called when a payment request is received by the
// Payment manager
type ReceivePaymentCallback func(address, amount, description string,
	paymentID PaymentID, sender *id.ID)

// PaymentStatusCallback is called when status updates are received over
// the payment receipt channel
type PaymentStatusCallback func(paymentID PaymentID, status Status)

// PaymentStatus struct represents the status messages received from processors
type PaymentStatus struct {
	id     PaymentID
	status Status
}

// PrivateTransaction is sent to a processor to handle an approved payment
type PrivateTransaction struct {
	SignedTransaction []byte
	Data              []byte
}

// PaymentInfo defines a payment request
type PaymentInfo struct {
	Address           string
	Amount            string
	Data              string
	Id                PaymentID
	XxnBlockchainDest string
}

type payment struct {
	sender            *id.ID
	recipient         *id.ID
	status            Status
	receiptChannel    broadcast.Channel
	receiptChannelKey *rsa.PrivateKey
	info              *PaymentInfo
}

type E2E interface {
	GetRng() *fastRNG.StreamGenerator
	GetE2E() e2e.Handler
	GetCmix() cmix.Client
	GetReceptionIdentity() xxdk.ReceptionIdentity
}
