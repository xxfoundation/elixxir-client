///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/elixxir/primitives/notifications"
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

type ManyNotificationForMeReport struct {
	many []*NotificationForMeReport
}

func (mnfmr *ManyNotificationForMeReport) Get(i int) (*NotificationForMeReport, error) {
	if i >= len(mnfmr.many) {
		return nil, errors.New("Cannot get, too long")
	}
	return mnfmr.many[i], nil
}

func (mnfmr *ManyNotificationForMeReport) Len() int {
	return len(mnfmr.many)
}

// NotificationsForMe Check if a notification received is for me
// It returns a NotificationForMeReport which contains a ForMe bool stating if it is for the caller,
// a Tag, and a source. These are as follows:
//	TYPE       	SOURCE				DESCRIPTION
// 	"default"	recipient user ID	A message with no association
//	"request"	sender user ID		A channel request has been received
//	"confirm"	sender user ID		A channel request has been accepted
//	"silent"	sender user ID		A message which should not be notified on
//	"e2e"		sender user ID		reception of an E2E message
//	"group"		group ID			reception of a group chat message
//  "endFT"     sender user ID		Last message sent confirming end of file transfer
//  "groupRQ"   sender user ID		Request from sender to join a group chat
func NotificationsForMe(notifCSV, preimages string) (*ManyNotificationForMeReport, error) {
	//handle deserialization of preimages
	var preimageList []edge.Preimage
	if err := json.Unmarshal([]byte(preimages), &preimageList); err != nil {
		return nil, errors.WithMessagef(err, "Failed to unmarshal the preimages list, "+
			"cannot check if notification is for me")
	}

	list, err := notifications.DecodeNotificationsCSV(notifCSV)

	if err != nil {
		return nil, err
	}

	notifList := make([]*NotificationForMeReport, 0, len(list))

	for _, notifData := range list {
		n := &NotificationForMeReport{
			forMe:  false,
			tYpe:   "",
			source: nil,
		}
		//check if any preimages match with the passed in data
		for _, preimage := range preimageList {
			if fingerprint.CheckIdentityFpFromMessageHash(notifData.IdentityFP, notifData.MessageHash, preimage.Data) {
				n = &NotificationForMeReport{
					forMe:  true,
					tYpe:   preimage.Type,
					source: preimage.Source,
				}
				break
			}
		}
		notifList = append(notifList, n)
	}

	return &ManyNotificationForMeReport{many: notifList}, nil
}

// RegisterForNotifications accepts firebase messaging token
func (c *Client) RegisterForNotifications(token string) error {
	return c.api.RegisterForNotifications(token)
}

// UnregisterForNotifications unregister user for notifications
func (c *Client) UnregisterForNotifications() error {
	return c.api.UnregisterForNotifications()
}
