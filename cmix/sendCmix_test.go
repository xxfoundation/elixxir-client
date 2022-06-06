package cmix

// func TestClient_SendCMIX(t *testing.T) {
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
// 	params := GetDefaultCMIXParams()
// 	rid, eid, err := c.Send(recipientID, fp, service, contents, mac, params)
// 	if err != nil {
// 		t.Errorf("Failed to sendcmix: %+v", err)
// 		t.FailNow()
// 	}
// 	t.Logf("Test of Send returned:\n\trid: %v\teid: %+v", rid, eid)
// }
