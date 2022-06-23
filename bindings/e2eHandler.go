////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/xx_network/primitives/id"
)

// IdList is a wrapper for a list of marshalled id.ID objects
type IdList struct {
	Ids [][]byte
}

// GetReceptionID returns the marshalled default IDs
func (e *E2e) GetReceptionID() []byte {
	return e.api.GetE2E().GetReceptionID().Marshal()
}

// GetAllPartnerIDs returns a marshalled list of all partner IDs that the user has
// an E2E relationship with.
func (e *E2e) GetAllPartnerIDs() ([]byte, error) {
	partnerIds := e.api.GetE2E().GetAllPartnerIDs()
	convertedIds := make([][]byte, len(partnerIds))
	for i, partnerId := range partnerIds {
		convertedIds[i] = partnerId.Marshal()
	}
	return json.Marshal(IdList{Ids: convertedIds})
}

// PayloadSize Returns the max payload size for a partitionable E2E
// message
func (e *E2e) PayloadSize() int {
	return int(e.api.GetE2E().PayloadSize())
}

// SecondPartitionSize returns the max partition payload size for all
// payloads after the first payload
func (e *E2e) SecondPartitionSize() int {
	return int(e.api.GetE2E().SecondPartitionSize())
}

// PartitionSize returns the partition payload size for the given
// payload index. The first payload is index 0.
func (e *E2e) PartitionSize(payloadIndex int) int {
	return int(e.api.GetE2E().PartitionSize(uint(payloadIndex)))
}

// FirstPartitionSize returns the max partition payload size for the
// first payload
func (e *E2e) FirstPartitionSize() int {
	return int(e.api.GetE2E().FirstPartitionSize())
}

// GetHistoricalDHPrivkey returns the user's marshalled Historical DH Private Key
func (e *E2e) GetHistoricalDHPrivkey() ([]byte, error) {
	return e.api.GetE2E().GetHistoricalDHPrivkey().MarshalJSON()
}

// GetHistoricalDHPubkey returns the user's marshalled Historical DH
// Public Key
func (e *E2e) GetHistoricalDHPubkey() ([]byte, error) {
	return e.api.GetE2E().GetHistoricalDHPubkey().MarshalJSON()
}

// HasAuthenticatedChannel returns true if an authenticated channel with the
// partner exists, otherwise returns false
func (e *E2e) HasAuthenticatedChannel(partnerId []byte) (bool, error) {
	partner, err := id.Unmarshal(partnerId)
	if err != nil {
		return false, err
	}
	return e.api.GetE2E().HasAuthenticatedChannel(partner), nil
}

// RemoveService removes all services for the given tag
func (e *E2e) RemoveService(tag string) error {
	return e.api.GetE2E().RemoveService(tag)
}

// SendE2E send a message containing the payload to the
// recipient of the passed message type, per the given
// parameters - encrypted with end-to-end encryption.
// Default parameters can be retrieved through
func (e *E2e) SendE2E(messageType int, recipientId, payload,
	e2eParams []byte) ([]byte, error) {

	params := e2e.GetDefaultParams()
	err := params.UnmarshalJSON(e2eParams)
	if err != nil {
		return nil, err
	}
	recipient, err := id.Unmarshal(recipientId)
	if err != nil {
		return nil, err
	}

	roundIds, messageId, ts, err := e.api.GetE2E().SendE2E(catalog.MessageType(messageType), recipient, payload, params)
	if err != nil {
		return nil, err
	}

	result := SendE2eResults{
		roundIds:  roundIds,
		messageId: messageId.Marshal(),
		ts:        ts.UnixNano(),
	}
	return json.Marshal(result)
}

// SendE2eResults is the return type for SendE2e
type SendE2eResults struct {
	roundIds  []id.Round
	messageId []byte
	ts        int64
}
