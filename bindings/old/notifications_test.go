package old

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/fingerprint"
	"math/rand"
	"testing"
)

func TestNotificationForMe(t *testing.T) {

	const numPreimages = 5

	types := []string{"default", "request", "silent", "e2e", "group"}
	sourceList := [][]byte{{0}, {1}, {2}, {3}, {4}}

	preimageList := make([]edge.Preimage, 0, numPreimages)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < numPreimages; i++ {
		piData := make([]byte, 32)
		rng.Read(piData)

		pi := edge.Preimage{
			Data:   piData,
			Type:   types[i],
			Source: sourceList[i],
		}

		preimageList = append(preimageList, pi)
	}

	preimagesJson, _ := json.Marshal(&preimageList)

	dataSources := []int{0, 1, -1, 2, 3, 4, -1, 0, 1, 2, 3, 4, -1, 2, 2, 2}

	notifData := make([]*mixmessages.NotificationData, 0, len(dataSources))

	for _, index := range dataSources {
		var preimage []byte
		if index == -1 {
			preimage = make([]byte, 32)
			rng.Read(preimage)
		} else {
			preimage = preimageList[index].Data
		}

		msg := make([]byte, 32)
		rng.Read(msg)
		msgHash := fingerprint.GetMessageHash(msg)

		identityFP := fingerprint.IdentityFP(msg, preimage)

		n := &mixmessages.NotificationData{
			EphemeralID: 0,
			IdentityFP:  identityFP,
			MessageHash: msgHash,
		}

		notifData = append(notifData, n)
	}

	notfsCSV := mixmessages.MakeNotificationsCSV(notifData)

	notifsForMe, err := NotificationsForMe(notfsCSV, string(preimagesJson))
	if err != nil {
		t.Errorf("Got error from NotificationsForMe: %+v", err)
	}

	for i := 0; i < notifsForMe.Len(); i++ {
		nfm, err := notifsForMe.Get(i)
		if err != nil {
			t.Errorf("Got error in getting notif: %+v", err)
		}
		if dataSources[i] == -1 {
			if nfm.ForMe() {
				t.Errorf("Notification %d should not be for me", i)
			}
			if nfm.Type() != "" {
				t.Errorf("Notification %d shoudl not have a type, "+
					"has: %s", i, nfm.Type())
			}
			if nfm.Source() != nil {
				t.Errorf("Notification %d shoudl not have a source, "+
					"has: %v", i, nfm.Source())
			}
		} else {
			if !nfm.ForMe() {
				t.Errorf("Notification %d should be for me", i)
			} else {
				expectedType := types[dataSources[i]]
				if nfm.Type() != expectedType {
					t.Errorf("Notification %d has the wrong type, "+
						"Expected: %s, Received: %s", i, nfm.Type(), expectedType)
				}
				expectedSource := sourceList[dataSources[i]]
				if !bytes.Equal(nfm.Source(), expectedSource) {
					t.Errorf("Notification %d source does not match: "+
						"Expected: %v, Received: %v", i, expectedSource,
						nfm.Source())
				}
			}
		}
	}
}

func TestManyNotificationForMeReport_Get(t *testing.T) {
	ManyNotificationForMeReport := &ManyNotificationForMeReport{many: make([]*NotificationForMeReport, 10)}

	//not too long
	_, err := ManyNotificationForMeReport.Get(2)
	if err != nil {
		t.Errorf("Got error when not too long: %+v", err)
	}

	//too long
	_, err = ManyNotificationForMeReport.Get(69)
	if err == nil {
		t.Errorf("Didnt get error when too long")
	}
}
