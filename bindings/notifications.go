////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/primitives/notifications"
)

// NotificationReports is a list of [NotificationReport]s. This will be returned
// via GetNotificationsReport as a JSON marshalled byte data.
//
// Example JSON:
//
//	[
//	  {
//	    "ForMe": true,                                           // boolean
//	    "Type": "e2e",                                           // string
//	    "Source": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD" // bytes of id.ID encoded as base64 string
//	  },
//	  {
//	    "ForMe": true,
//	    "Type": "e2e",
//	    "Source": "AAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"
//	  },
//	  {
//	    "ForMe": true,
//	    "Type": "e2e",
//	    "Source": "AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"
//	  }
//	]
type NotificationReports []NotificationReport

//  TODO: The table in the docstring below needs to be checked for completeness
//   and/or accuracy to ensure descriptions/sources are still accurate (they are
//   from the old implementation).

// NotificationReport is the bindings' representation for notifications for
// this user.
//
// Example NotificationReport JSON:
//
//		{
//		  "ForMe": true,
//	   "Type": [
//	      "e2e"
//	   ],
//		  "Source": "dGVzdGVyMTIzAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
//		}
//
// Given the Type, the Source value will have specific contextual meanings.
// Below is a table that will define the contextual meaning of the Source field
// given all possible Type fields.
//
//	 TYPE     |     SOURCE         |    DESCRIPTION
//	----------+--------------------+--------------------------------------------------------
//	"default" |  recipient user ID |  A message with no association.
//	"request" |  sender user ID    |  A channel request has been received, from Source.
//	"reset"   |  sender user ID    |  A channel reset has been received.
//	"confirm" |  sender user ID    |  A channel request has been accepted.
//	"silent"  |  sender user ID    |  A message where the user should not be notified.
//	"e2e"     |  sender user ID    |  A reception of an E2E message.
//	"group"   |  group ID          |  A reception of a group chat message.
//	"endFT"   |  sender user ID    |  The last message sent confirming end of file transfer.
//	"groupRQ" |  sender user ID    |  A request from Source to join a group chat.
type NotificationReport struct {
	// ForMe determines whether this value is for the user. If it is
	// false, this report may be ignored.
	ForMe bool
	// Type is the type of notification. The list can be seen
	Type []string
	// Source is the source of the notification.
	Source []byte
}

// GetNotificationsReport parses the received notification data to determine which
// notifications are for this user. // This returns the JSON-marshalled
// NotificationReports.
//
// Parameters:
//   - notificationCSV - the notification data received from the
//     notifications' server.
//   - marshalledServices - the JSON-marshalled list of services the backend
//     keeps track of. Refer to Cmix.TrackServices or
//     Cmix.TrackServicesWithIdentity for information about this.
//
// Returns:
//   - []byte - A JSON marshalled NotificationReports. Some NotificationReport's
//     within in this structure may have their NotificationReport.ForMe
//     set to false. These may be ignored.
func GetNotificationsReport(notificationCSV string,
	marshalledServices, marshalledCompressedServices []byte) ([]byte, error) {

	// Decode notifications' server data
	notificationList, err := notifications.DecodeNotificationsCSV(notificationCSV)
	if err != nil {
		return nil, err
	}

	// Parse notifications list against all marshalled services
	servicesReport, err := getServicesReport(
		marshalledServices, notificationList)
	if err != nil {
		return nil, err
	}

	// Parse notifications list against all marshalled compressed services
	compressedServicesReport, err := getCompressedServicesReport(
		marshalledCompressedServices, notificationList)
	if err != nil {
		return nil, err
	}

	// Concatenate these reports
	reportList := append(servicesReport, compressedServicesReport...)

	return json.Marshal(reportList)
}

// RegisterForNotifications allows a client to register for push notifications.
// The token is a firebase messaging token.
//
// Parameters:
//   - e2eId - ID of the E2E object in the E2E tracker
func RegisterForNotifications(e2eId int, token string) error {
	user, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return err
	}

	return user.api.RegisterForNotifications(user.api.GetReceptionIdentity().ID, token)
}

// UnregisterForNotifications turns off notifications for this client.
//
// Parameters:
//   - e2eId - ID of the E2E object in the E2E tracker
func UnregisterForNotifications(e2eId int) error {
	user, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return err
	}

	return user.api.UnregisterForNotifications(user.api.GetReceptionIdentity().ID)
}

// getCompressedServicesReport is a utility function for
// [GetNotificationsReport].
//
// getCompressedServicesReport parses the notifications list against the
// marshalled compressed services. If a compressed service determines a
// notification is meant for this user, it appends this data to the list of
// [NotificationReport]'s.
func getCompressedServicesReport(marshalledCompressedServices []byte,
	notificationList []*notifications.Data) ([]*NotificationReport, error) {
	// Construct  a report list
	reportList := make([]*NotificationReport, len(notificationList))

	// Process compressed services
	compressServiceList := message.CompressedServiceList{}
	err := json.Unmarshal(marshalledCompressedServices, &compressServiceList)
	if err != nil {
		return nil, err
	}

	for id, services := range compressServiceList {
		// Iterate over potential notifications for us
		for i := range notificationList {
			notifData := notificationList[i]

			// Iterate over all services {
			for j := range services {
				service := services[j]
				found, tags, metadata := service.ForMe(&id,
					notifData.MessageHash, service.Metadata)
				if !found {
					continue
				}

				reportList[i] = &NotificationReport{
					ForMe:  true,
					Type:   tags,
					Source: metadata,
				}

			}
		}
	}

	return reportList, nil
}

// getServicesReport is a utility function for [GetNotificationsReport].
//
// getServicesReport parses the notifications list against the marshalled
// services. If a compressed service determines a notification is meant for this
// user, it appends this data to the list of [NotificationReport]'s.
func getServicesReport(marshalledServices []byte,
	notificationList []*notifications.Data) ([]*NotificationReport, error) {
	// Construct  a report list
	reportList := make([]*NotificationReport, 0)

	// If services are retrieved using TrackServicesWithIdentity, this
	// should return a single list.
	serviceList := message.ServiceList{}
	err := json.Unmarshal(marshalledServices, &serviceList)
	if err != nil {
		return nil, err
	}

	// Iterate over data provided by server
	for _, services := range serviceList {
		for i := range notificationList {
			notifData := notificationList[i]

			// Iterate over all services
			for j := range services {
				// Pull data from services and from notification data
				service := services[j]
				hash := service.HashFromMessageHash(notifData.MessageHash)

				// Check if this notification data is recognized by
				// this service, ie "ForMe"
				if service.ForMeFromMessageHash(notifData.MessageHash, hash) {
					// Fill report list with service data
					reportList[i] = &NotificationReport{
						ForMe:  true,
						Type:   []string{service.Tag},
						Source: service.Identifier,
					}
				}
			}
		}
	}

	return reportList, nil
}
