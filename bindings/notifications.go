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


type NotificationForMeReport struct{
	forMe bool
	tYpe string
	source []byte
}

func (nfmr *NotificationForMeReport)ForMe()bool{
	return nfmr.forMe
}

func (nfmr *NotificationForMeReport)Type()string{
	return nfmr.tYpe
}

func (nfmr *NotificationForMeReport)Source()[]byte{
	return nfmr.source
}


// NotificationForMe Check if a notification received is for me
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
	var preimageList edge.Preimages
	if err := json.Unmarshal([]byte(preimages),&preimageList); err!=nil{
		return nil, errors.WithMessagef(err,"Failed to unmarshal the preimages list, " +
			"cannot check if notification is for me")
	}

	//check if any preimages match with the passed in data
	for i:=0;i<len(preimageList);i++{
		if fingerprint.CheckIdentityFpFromMessageHash(idFpBytes, messageHashBytes, preimageList[i].Data){
			return &NotificationForMeReport{
				forMe:  true,
				tYpe:   preimageList[i].Type,
				source: preimageList[i].Source,
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
