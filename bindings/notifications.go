///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/elixxir/crypto/fingerprint"
)

type NotificationForMeReport struct {
	forMe  bool
	tYpe   string
	source []byte
}

func (nfmr *NotificationForMeReport) ForMe() bool {
	return nfmr.forMe
}

func (nfmr *NotificationForMeReport) Type() string {
	return nfmr.tYpe
}

func (nfmr *NotificationForMeReport) Source() []byte {
	return nfmr.source
}

// NotificationForMe Check if a notification received is for me
// It returns a NotificationForMeReport which contains a ForMe bool stating if it is for the caller,
// a Type, and a source. These are as follows:
//	TYPE       	DESCRIPTION							SOURCE
// 	"default"	A message with no association		recipient user ID
//	"request"	A channel request has been received recipient user ID
//	"confirm"	A channel request has been accepted sender user ID
//	"rekey"		keys with a user have been rotated	sender user ID
//	"e2e"		reception of an E2E message			sender user ID
//	"group"		eception of a group chat message	group ID
func NotificationForMe(messageHash, idFP string, preimages string) (*NotificationForMeReport, error) {
	//handle message hash and idFP
	messageHashBytes, err := base64.StdEncoding.DecodeString(messageHash)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to decode message ID")
	}
	idFpBytes, err := base64.StdEncoding.DecodeString(idFP)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to decode identity fingerprint")
	}

	//handle deserialization of preimages
	var preimageList []edge.Preimage
	if err := json.Unmarshal([]byte(preimages), &preimageList); err != nil {
		return nil, errors.WithMessagef(err, "Failed to unmarshal the preimages list, "+
			"cannot check if notification is for me")
	}

	//check if any preimages match with the passed in data
	for _, preimage := range preimageList {
		if fingerprint.CheckIdentityFpFromMessageHash(idFpBytes, messageHashBytes, preimage.Data) {
			return &NotificationForMeReport{
				forMe:  true,
				tYpe:   preimage.Type,
				source: preimage.Source,
			}, nil
		}
	}
	return &NotificationForMeReport{
		forMe:  false,
		tYpe:   "",
		source: nil,
	}, nil
}

// RegisterForNotifications accepts firebase messaging token
func (c *Client) RegisterForNotifications(token string) error {
	return c.api.RegisterForNotifications(token)
}

// UnregisterForNotifications unregister user for notifications
func (c *Client) UnregisterForNotifications() error {
	return c.api.UnregisterForNotifications()
}
