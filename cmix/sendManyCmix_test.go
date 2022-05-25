package cmix

// func TestClient_SendMany_SendManyCMIX(t *testing.T) {
// 	c, err := newTestClient(t)
// 	if err != nil {
// 		t.Fatalf("Failed to create test client: %+v", err)
// 	}

// 	recipientID := id.NewIdFromString("zezima", id.User, t)
// 	contents := []byte("message")
// 	fp := format.NewFingerprint(contents)
// 	service := message.GetDefaultService(recipientID)
// 	mac := make([]byte, 32)
// 	_, err = csprng.NewSystemRNG().Read(mac)
// 	if err != nil {
// 		t.Errorf("Failed to read random mac bytes: %+v", err)
// 	}
// 	mac[0] = 0
// 	messages := []TargetedCmixMessage{
// 		{
// 			Recipient:   recipientID,
// 			Payload:     contents,
// 			Fingerprint: fp,
// 			Service:     service,
// 			Mac:         mac,
// 		},
// 		{
// 			Recipient:   recipientID,
// 			Payload:     contents,
// 			Fingerprint: fp,
// 			Service:     service,
// 			Mac:         mac,
// 		},
// 	}

// 	rid, eid, err := c.SendMany(messages, GetDefaultCMIXParams())
// 	if err != nil {
// 		t.Errorf("Failed to run SendMany: %+v", err)
// 	}
// 	t.Logf("Test of SendMany returned:\n\trid: %v\teid: %+v", rid, eid)

// }
