package noobChannel

import (
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
	"time"
)

func TestManager_GetNoobChannel(t *testing.T) {
	storageDir := "/tmp/clientTestStorage"
	ndfPath := "/tmp/ndf.json"
	ndfBytes, err := utils.ReadFile(ndfPath)
	ndfJson := string(ndfBytes)
	password := []byte("password")
	if !utils.Exists(storageDir) {
		err := xxdk.NewCmix(ndfJson, storageDir, password, "")
		if err != nil {
			t.Fatal(err)
		}
	}

	cmix, err := xxdk.LoadCmix(storageDir, password, xxdk.GetDefaultCMixParams())
	if err != nil {
		t.Fatal(err)
	}

	contactPath := "/tmp/contact.json"
	contactBytes, err := utils.ReadFile(contactPath)
	if err != nil {
		t.Fatal(err)
	}
	ncContact, err := contact.Unmarshal(contactBytes)
	if err != nil {
		t.Fatal(err)
	}

	rid, err := xxdk.MakeReceptionIdentity(cmix)
	if err != nil {
		t.Fatal(err)
	}

	e2eClient, err := xxdk.Login(cmix, xxdk.DefaultAuthCallbacks{}, rid, xxdk.GetDefaultE2EParams())
	if err != nil {
		t.Fatal(err)
	}

	err = e2eClient.StartNetworkFollower(time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	for isReady := false; !isReady; {
		time.Sleep(time.Second)
		var progress float64
		e2eClient.NetworkFollowerStatus()
		isReady, progress = e2eClient.IsReady(0.65)
		t.Log(progress)
	}

	receivedChannel, err := GetNoobChannel(e2eClient, ncContact)

	t.Log(receivedChannel)
}
