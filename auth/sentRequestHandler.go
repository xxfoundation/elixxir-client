package auth

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/cmix/message"
)

// sentRequestHandler interface which allows the lower level to register
// sent requests with cmix while blackboxing it
type sentRequestHandler struct {
	s *State
}

// Add Adds the service and fingerprints to cmix for the given sent request
func (srh *sentRequestHandler) Add(sr *store.SentRequest) {
	fp := sr.GetFingerprint()
	rc := &receivedConfirmService{
		s:           srh.s,
		SentRequest: sr,
		notificationsService: message.Service{
			Identifier: fp[:],
			Tag:        srh.s.params.getConfirmTag(sr.IsReset()),
			Metadata:   nil,
		},
	}

	//add the notifications service
	srh.s.net.AddService(srh.s.e2e.GetReceptionID(), rc.notificationsService, nil)

	//add the fingerprint
	if err := srh.s.net.AddFingerprint(srh.s.e2e.GetReceptionID(),
		sr.GetFingerprint(), rc); err != nil {
		jww.FATAL.Panicf("failed to add a fingerprint for a auth confirm, " +
			"this should never happen under the birthday paradox assumption of " +
			"255 bits (the size fo the fingerprint).")
	}

}

// Delete removes the service and fingerprints to cmix for the given sent
// request
func (srh *sentRequestHandler) Delete(sr *store.SentRequest) {
	fp := sr.GetFingerprint()

	notificationsService := message.Service{
		Identifier: fp[:],
		Tag:        srh.s.params.getConfirmTag(sr.IsReset()),
		Metadata:   nil,
	}

	//delete the notifications service
	srh.s.net.DeleteService(srh.s.e2e.GetReceptionID(), notificationsService, nil)

	// delete the fingerprint, this will do nothing if the delete was caused by
	// usage, but it will delete if this is a deletion of the request
	srh.s.net.DeleteFingerprint(srh.s.e2e.GetReceptionID(), sr.GetFingerprint())
}
