package payments

import (
	"crypto/rsa"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoBackup "gitlab.com/elixxir/crypto/backup"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	e2e2 "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const (
	paymentsPrefix         = "payments"
	pendingRequestsKey     = "pendingRequests"
	pendingRequestsVersion = uint64(0)
)

// manager handles the logic & storage for payments sent & received
type manager struct {
	e2e E2E
	kv  *versioned.KV

	requestsLock    *sync.Mutex
	pendingRequests map[PaymentID]*payment

	receiveCallback ReceivePaymentCallback
	statusCallback  PaymentStatusCallback
}

// NewOrLoad returns a Payments interface, loading from storage if it exists,
// or initializing new payments data in storage if not.
func NewOrLoad(rkv *versioned.KV, e E2E,
	receiveCallback ReceivePaymentCallback,
	statusCallback PaymentStatusCallback) (Payments, error) {
	kv := rkv.Prefix(paymentsPrefix)

	pm := &manager{
		e2e:             e,
		kv:              kv,
		receiveCallback: receiveCallback,
		statusCallback:  statusCallback,
		requestsLock:    &sync.Mutex{},
	}

	// Load pending requests, or create it if not stored
	if err := pm.loadPendingRequests(); err != nil {
		pm.pendingRequests = make(map[PaymentID]*payment)
		err = pm.savePendingRequests()
		if err != nil {
			return nil, errors.WithMessage(err,
				"Failed to init new payments store pending requests")
		}
	}

	// Start listening for status updates on pending requests
	for _, req := range pm.pendingRequests {
		switch req.status {
		case Sent, Pending, Submitted:
			err := pm.waitOnPayment(req)
			if err != nil {
				return nil, errors.WithMessage(err,
					"failed to start payment status channel")
			}
		}
	}

	// Register listener for incoming payment requests
	pm.e2e.GetE2E().RegisterListener(&id.ZeroUser, catalog.PaymentRequest,
		&listener{m: pm})

	return pm, nil
}

// Request sends a request for payment to the passed in recipient, creates the
// channel based on the Key Residue from SendE2E, and starts a listener
// on the receipt channel for status updates
func (pm *manager) Request(amount, address, description,
	xxnBlockchainDestination string, recipient *id.ID) (PaymentID, error) {
	// Create a payment ID
	paymentId := PaymentID{}
	_, err := pm.e2e.GetRng().GetStream().Read(paymentId[:])
	if err != nil {
		return PaymentID{}, err
	}

	// Make payment info and marshal to bytes for payload
	payload := &PaymentInfo{
		Address:           address,
		Amount:            amount,
		Data:              description,
		Id:                paymentId,
		XxnBlockchainDest: xxnBlockchainDestination,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return PaymentID{}, err
	}

	report, err := pm.e2e.GetE2E().SendE2E(catalog.PaymentRequest, recipient,
		payloadBytes, e2e.GetDefaultParams())
	if err != nil {
		return PaymentID{}, err
	}

	// Create channel based on key residue from e2e send
	receiptChannel, key, err := makeChannelForRequest(report.KeyResidue, payload)

	// Add payment to storage and start listening for status updates
	p := &payment{
		sender:            pm.e2e.GetReceptionIdentity().ID,
		status:            Sent,
		recipient:         recipient,
		receiptChannel:    receiptChannel,
		receiptChannelKey: key,
		info:              payload,
	}
	pm.requestsLock.Lock()
	pm.pendingRequests[paymentId] = p
	pm.requestsLock.Unlock()

	err = pm.savePendingRequests()
	if err != nil {
		return PaymentID{}, err
	}

	err = pm.waitOnPayment(p)

	return paymentId, nil
}

// Approve sends payment approval to the passed in xx network destination for
// processing, and starts a listener on the receipt channel for payment status
// updates
func (pm *manager) Approve(signedTransaction, data []byte, paymentId PaymentID,
	xxnetworkBlockchainDestination *id.ID) error {
	pt := &PrivateTransaction{
		// TODO do we need to send the channel along with this?
		SignedTransaction: signedTransaction,
		Data:              data,
	}
	payloadBytes, err := json.Marshal(pt)
	if err != nil {
		return err
	}
	// TODO docs say this can be sent without encryption, does this mean use sendunsafe?
	_, _, err = pm.e2e.GetE2E().SendUnsafe(catalog.PaymentConf,
		xxnetworkBlockchainDestination, payloadBytes, e2e.GetDefaultParams())
	if err != nil {
		return err
	}

	// Update stored payment status
	pm.requestsLock.Lock()
	pm.pendingRequests[paymentId].status = Pending
	pm.requestsLock.Unlock()

	err = pm.savePendingRequests()
	if err != nil {
		return err
	}

	// Start listening for status updates
	err = pm.waitOnPayment(pm.pendingRequests[paymentId])
	if err != nil {
		return err
	}
	return nil
}

// GetRequest returns a PaymentRequest interface for the passed in ID
func (pm *manager) GetRequest(id PaymentID) (PaymentRequest, bool) {
	req, ok := pm.pendingRequests[id]
	return req, ok
}

func (pm *manager) GetRequests() (reqs []PaymentRequest) {
	for _, val := range pm.pendingRequests {
		reqs = append(reqs, val)
	}
	return
}

// waitOnPayment creates a broadcast channel client from the receipt channel
// and starts a listener for status updates which both updates the status in
// the manager & calls the status update callback
func (pm *manager) waitOnPayment(request *payment) error {
	// Create broadcast channel handler
	bcm, err := broadcast.NewBroadcastChannel(request.receiptChannel,
		pm.e2e.GetCmix(), pm.e2e.GetRng())
	if err != nil {
		return err
	}

	// Define listener for the payment receipt channel to handle status updates
	l := func(payload []byte,
		receptionID receptionID.EphemeralIdentity, round rounds.Round) {
		// TODO is this the correct decryption method?
		rng := pm.e2e.GetRng().GetStream()
		decrypted, err := request.receiptChannelKey.Decrypt(rng,
			payload, cryptoBackup.DefaultParams())
		if err != nil {
			jww.ERROR.Printf("Failed to decrypt payment status "+
				"update: %+v", err)
		}
		// Unmarshal received payment status
		statusMessage := &PaymentStatus{}
		err = json.Unmarshal(decrypted, statusMessage)
		if err != nil {
			jww.ERROR.Printf("Failed to unmarshal payment status: %+v",
				err)
		}

		// Call status callback
		go pm.statusCallback(statusMessage.id, statusMessage.status)

		// Update stored status
		pm.requestsLock.Lock()
		if statusMessage.status == Accepted {
			// TODO do we want to do this automatically or have another call for it?
			delete(pm.pendingRequests, statusMessage.id)
		} else {
			pm.pendingRequests[statusMessage.id].status = statusMessage.status
		}
		pm.requestsLock.Unlock()

		err = pm.savePendingRequests()
		if err != nil {
			jww.ERROR.Printf("Failed to save pending requests: %+v", err)
		}
	}

	// Register listener on symmetric messages
	err = bcm.RegisterListener(l, broadcast.Symmetric)
	if err != nil {
		return errors.WithMessage(err, "Failed to add listener"+
			" for payment status")
	}
	return nil
}

// Loads pending requests from storage
func (pm *manager) loadPendingRequests() error {
	pm.requestsLock.Lock()
	defer pm.requestsLock.Unlock()
	obj, err := pm.kv.Get(pendingRequestsKey, pendingRequestsVersion)
	if err != nil {
		return err
	}
	err = json.Unmarshal(obj.Data, &pm.pendingRequests)
	if err != nil {
		return err
	}
	return nil
}

// Saves pending requests in storage
func (pm *manager) savePendingRequests() error {
	marshalled, err := json.Marshal(pm.pendingRequests)
	if err != nil {
		return err
	}
	err = pm.kv.Set(pendingRequestsKey,
		&versioned.Object{
			Version:   pendingRequestsVersion,
			Timestamp: time.Now(),
			Data:      marshalled,
		})
	if err != nil {
		return err
	}
	return nil
}

func makeChannelForRequest(residue e2e2.KeyResidue,
	info *PaymentInfo) (cryptoBroadcast.Channel, *rsa.PrivateKey, error) {
	// TODO create channel based on e2e key residue
	return cryptoBroadcast.Channel{}, nil, nil
}

func (p *payment) GetInfo() *PaymentInfo {
	return p.info
}

func (p *payment) GetStatus() Status {
	return p.status
}
