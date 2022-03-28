package message

import (
	"gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/format"
)

type Processor interface {
	// Process decrypts and hands off the message to its internal down stream
	// message processing system.
	// CRITICAL: Fingerprints should never be used twice. Process must denote,
	// in long term storage, usage of a fingerprint and that fingerprint must
	// not be added again during application load.
	// It is a security vulnerability to reuse a fingerprint. It leaks privacy
	// and can lead to compromise of message contents and integrity.
	Process(message format.Message, receptionID receptionID.EphemeralIdentity,
		round *mixmessages.RoundInfo)
}
