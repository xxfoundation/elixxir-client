///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/base64"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/xx_network/primitives/id"
)

// NotificationForMe Check if a notification received is for me
func NotificationForMe(messageHash, idFP string, receptionId []byte) (bool, error) {
	messageHashBytes, err := base64.StdEncoding.DecodeString(messageHash)
	if err != nil {
		return false, errors.WithMessage(err, "Failed to decode message ID")
	}
	idFpBytes, err := base64.StdEncoding.DecodeString(idFP)
	if err != nil {
		return false, errors.WithMessage(err, "Failed to decode identity fingerprint")
	}
	rid, err := id.Unmarshal(receptionId)
	if err != nil {
		return false, errors.WithMessage(err, "Failed to unmartial reception ID")
	}
	return fingerprint.CheckIdentityFpFromMessageHash(idFpBytes, messageHashBytes, rid), nil
}

// RegisterForNotifications accepts firebase messaging token
func (c *Client) RegisterForNotifications(token []byte) error {
	return c.api.RegisterForNotifications(token)
}

// UnregisterForNotifications unregister user for notifications
func (c *Client) UnregisterForNotifications() error {
	return c.api.UnregisterForNotifications()
}
