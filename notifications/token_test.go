package notifications

import (
	"errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	notifCrypto "gitlab.com/elixxir/crypto/notifications"
	"testing"
	"time"
)

func TestManager_AddToken_Smoke(t *testing.T) {

	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	testToken := "mickey"
	testApp := "mouse"

	err := m.AddToken(testToken, testApp)
	if err != nil {
		t.Errorf("Failed to add token when it should be "+
			"possible: %+v", err)
	}

	if mInternal.token.Token != testToken || mInternal.token.App != testApp {
		t.Errorf("Tokens do not match after being set")
	}

	// check the message
	message := comms.receivedMessage.(*pb.RegisterTokenRequest)
	if message.Token != testToken || message.App != testApp {
		t.Errorf("Tokens do not match in the sent payload")
	}

	err = notifCrypto.VerifyToken(mInternal.transmissionRSA.Public(), testToken,
		testApp, time.Unix(0, message.RequestTimestamp),
		notifCrypto.RegisterTokenTag, message.TokenSignature)
	if err != nil {
		t.Errorf("Failed to verify the token unregister signature: %+v",
			err)
	}
}

func TestManager_RemoveToken_Smoke(t *testing.T) {

	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	testToken := "mickey"
	testApp := "mouse"

	//test without a token, it should error
	err := m.RemoveToken()
	if err == nil || !errors.Is(err, ErrNoTokenRegistered) {
		t.Errorf("remove token did not error when it should have: %+v", err)
	}

	//add a token
	err = m.AddToken(testToken, testApp)
	if err != nil {
		t.Errorf("Failed to add token when it should be "+
			"possible: %+v", err)
	}

	if mInternal.token.Token != testToken || mInternal.token.App != testApp {
		t.Errorf("Tokens do not match after being set")
	}

	// remove a token
	err = m.RemoveToken()
	if err != nil {
		t.Errorf("remove token errored when it shouldnt have: %+v", err)
	}

	if mInternal.token.Token != "" || mInternal.token.App != "" {
		t.Errorf("Tokens should be empty")
	}

	// check the message
	message := comms.receivedMessage.(*pb.UnregisterTokenRequest)
	if message.Token != testToken || message.App != testApp {
		t.Errorf("Tokens do not match in the sent payload")
	}

	err = notifCrypto.VerifyToken(mInternal.transmissionRSA.Public(), testToken,
		testApp, time.Unix(0, message.RequestTimestamp),
		notifCrypto.UnregisterTokenTag, message.TokenSignature)
	if err != nil {
		t.Errorf("Failed to verify the token unregister signature: %+v",
			err)
	}
}

func TestManager_GetToken_Smoke(t *testing.T) {

	m, _, _ := buildTestingManager(t)

	testToken := "mickey"
	testApp := "mouse"

	exist1, token1, app1 := m.GetToken()

	if exist1 {
		t.Errorf("token exists when it shouldnt")
	}

	if token1 != "" || app1 != "" {
		t.Errorf("token values are set exists when they shouldnt be")
	}

	err := m.AddToken(testToken, testApp)
	if err != nil {
		t.Errorf("Failed to add token when it should be "+
			"possible: %+v", err)
	}

	exist2, token2, app2 := m.GetToken()

	if !exist2 {
		t.Errorf("token doesnt exists when it shouldnt")
	}

	if token2 != testToken || app2 != testApp {
		t.Errorf("token values are not set correctly")
	}

	err = m.RemoveToken()
	if err != nil {
		t.Errorf("remove token errored when it shouldnt have: %+v", err)
	}

	exist3, token3, app3 := m.GetToken()

	if exist3 {
		t.Errorf("token  exists when it shouldnt")
	}

	if token3 != "" || app3 != "" {
		t.Errorf("token values are not set correctly")
	}
}
