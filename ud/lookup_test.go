package ud

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"strconv"
	"testing"
	"time"
)

// Happy path.
func TestManager_Lookup(t *testing.T) {
	// Set up mock UD values
	grp := getGroup()
	prng := NewPrng(42)
	privKeyBytes, err := csprng.GenerateInGroup(
		grp.GetP().Bytes(), grp.GetP().ByteLen(), prng)
	if err != nil {
		t.Fatalf("Failed to generate a mock private key: %v", err)
	}
	udMockPrivKey := grp.NewIntFromBytes(privKeyBytes)
	publicKey := grp.ExpG(udMockPrivKey, grp.NewInt(1))

	// Generate callback function
	callbackChan := make(mockChannel)
	callback := func(c contact.Contact, err error) {
		callbackChan <- mockResponse{
			c:   []contact.Contact{c},
			err: err,
		}
	}

	// Set up mock manager
	m, tnm := newTestManager(t)

	uid := id.NewIdFromUInt(0x500000000000000, id.User, t)
	expectedContact := contact.Contact{
		ID:       uid,
		DhPubKey: publicKey,
	}

	contacts := []*Contact{{
		UserID: expectedContact.ID.Bytes(),
		PubKey: expectedContact.DhPubKey.Bytes(),
	}}

	receiver := newMockReceiver(callbackChan, contacts, t)

	udbId, err := id.Unmarshal(tnm.instance.GetFullNdf().Get().UDB.ID)
	if err != nil {
		t.Fatalf("Failed to unmarshal ID in mock ndf: %v", err)
	}

	mockListener := single.Listen(LookupTag, udbId, udMockPrivKey,
		tnm, grp, receiver)

	defer mockListener.Stop()

	r := m.user.GetE2E().GetGroup().NewInt(1)
	m.user.GetE2E().GetGroup().Random(r)
	s := ""
	jsonable, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	for _, b := range jsonable {
		s += strconv.Itoa(int(b)) + ", "
	}

	t.Logf("%v", r.Bytes())
	t.Logf("%s", s)

	timeout := 500 * time.Millisecond
	p := single.RequestParams{
		Timeout:             timeout,
		MaxResponseMessages: 1,
		CmixParams:          cmix.GetDefaultCMIXParams(),
	}

	// Run the lookup
	_, _, err = Lookup(m.user, m.GetContact(), callback, uid, p)
	if err != nil {
		t.Errorf("Lookup() returned an error: %+v", err)
	}

	// Verify the callback is called
	select {
	case cb := <-callbackChan:
		if cb.err != nil {
			t.Errorf("Callback returned an error: %+v", cb.err)
		}

		if !reflect.DeepEqual([]contact.Contact{expectedContact},
			cb.c) {
			t.Errorf("Failed to get expected Contact."+
				"\n\texpected: %v\n\treceived: %v", expectedContact, cb.c)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Callback not called.")
	}
}
