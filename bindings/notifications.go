////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package bindings

// FIXME: This is the old NotificationsForMe code that needs to be fixed
/*
type NotificationForMeReport struct {
	ForMe  bool
	Type   string
	Source []byte
}

type ManyNotificationForMeReport struct {
	Many []*NotificationForMeReport
}

// NotificationsForMe Check if a notification received is for me
// It returns a NotificationForMeReport which contains a ForMe bool stating if it is for the caller,
// a Type, and a source. These are as follows:
//	TYPE       	SOURCE				DESCRIPTION
// 	"default"	recipient user ID	A message with no association
//	"request"	sender user ID		A channel request has been received
//	"reset"	    sender user ID		A channel reset has been received
//	"confirm"	sender user ID		A channel request has been accepted
//	"silent"	sender user ID		A message which should not be notified on
//	"e2e"		sender user ID		reception of an E2E message
//	"group"		group ID			reception of a group chat message
//  "endFT"     sender user ID		Last message sent confirming end of file transfer
//  "groupRQ"   sender user ID		Request from sender to join a group chat
func NotificationsForMe(notifCSV, preimages string) (*ManyNotificationForMeReport, error) {
	// Handle deserialization of preimages
	var preimageList []edge.Preimage
	if err := json.Unmarshal([]byte(preimages), &preimageList); err != nil {
		return nil, errors.WithMessagef(err, "Failed to unmarshal the " +
			"preimages list, cannot check if notification is for me")
	}

	list, err := notifications.DecodeNotificationsCSV(notifCSV)
	if err != nil {
		return nil, err
	}

	notifList := make([]*NotificationForMeReport, len(list))

	for i, notifData := range list {
		notifList[i] = &NotificationForMeReport{
			ForMe:  false,
			Type:   "",
			Source: nil,
		}
		// check if any preimages match with the passed in data
		for _, preimage := range preimageList {
			if fingerprint.CheckIdentityFpFromMessageHash(notifData.IdentityFP, notifData.MessageHash, preimage.Data) {
				notifList[i] = &NotificationForMeReport{
					ForMe:  true,
					Type:   preimage.Type,
					Source: preimage.Source,
				}
				break
			}
		}
	}

	return &ManyNotificationForMeReport{notifList}, nil
}*/

// RegisterForNotifications allows a client to register for push notifications.
// The token is a firebase messaging token.
//
// Parameters:
//  - e2eId - ID of the E2E object in the E2E tracker
func RegisterForNotifications(e2eId int, token string) error {
	user, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return err
	}

	return user.api.RegisterForNotifications(token)
}

// UnregisterForNotifications turns off notifications for this client.
//
// Parameters:
//  - e2eId - ID of the E2E object in the E2E tracker
func UnregisterForNotifications(e2eId int) error {
	user, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return err
	}

	return user.api.UnregisterForNotifications()
}
