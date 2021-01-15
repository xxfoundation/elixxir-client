package single

import (
	"bytes"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
)

// Happy path.
func Test_newTransmitMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	pubKeySize := prng.Intn(externalPayloadSize)
	expected := transmitMessage{
		data:    make([]byte, externalPayloadSize),
		pubKey:  make([]byte, pubKeySize),
		payload: make([]byte, externalPayloadSize-pubKeySize),
	}

	m := newTransmitMessage(externalPayloadSize, pubKeySize)

	if !reflect.DeepEqual(expected, m) {
		t.Errorf("newTransmitMessage() did not produce the expected transmitMessage."+
			"\nexpected: %#v\nreceived: %#v", expected, m)
	}
}

// Error path: public key size is larger than external payload size.
func Test_newTransmitMessage_PubKeySizeError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("newTransmitMessage() did not panic when the size of " +
				"the payload is smaller than the size of the public key.")
		}
	}()

	_ = newTransmitMessage(5, 10)
}

// Happy path.
func Test_mapTransmitMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	pubKeySize := prng.Intn(externalPayloadSize)
	pubKey := make([]byte, pubKeySize)
	prng.Read(pubKey)
	payload := make([]byte, externalPayloadSize-pubKeySize)
	prng.Read(payload)
	var data []byte
	data = append(data, pubKey...)
	data = append(data, payload...)
	m := mapTransmitMessage(data, pubKeySize)

	if !bytes.Equal(data, m.data) {
		t.Errorf("mapTransmitMessage() failed to map the correct bytes for data."+
			"\nexpected: %+v\nreceived: %+v", data, m.data)
	}

	if !bytes.Equal(pubKey, m.pubKey) {
		t.Errorf("mapTransmitMessage() failed to map the correct bytes for pubKey."+
			"\nexpected: %+v\nreceived: %+v", pubKey, m.pubKey)
	}

	if !bytes.Equal(payload, m.payload) {
		t.Errorf("mapTransmitMessage() failed to map the correct bytes for payload."+
			"\nexpected: %+v\nreceived: %+v", payload, m.payload)
	}
}

// Happy path.
func TestTransmitMessage_Marshal_Unmarshal(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	pubKeySize := prng.Intn(externalPayloadSize)
	data := make([]byte, externalPayloadSize)
	prng.Read(data)
	m := mapTransmitMessage(data, pubKeySize)

	msgBytes := m.Marshal()

	newMsg, err := unmarshalTransmitMessage(msgBytes, pubKeySize)
	if err != nil {
		t.Errorf("unmarshalTransmitMessage produced an error: %+v", err)
	}

	if !reflect.DeepEqual(m, newMsg) {
		t.Errorf("Failed to marshal/unmarshal message."+
			"\nexpected: %+v\nreceived: %+v", m, newMsg)
	}
}

// Error path: public key size is larger than byte slice.
func Test_unmarshalTransmitMessage_PubKeySizeError(t *testing.T) {
	_, err := unmarshalTransmitMessage([]byte{1, 2, 3}, 5)
	if err == nil {
		t.Error("unmarshalTransmitMessage() did not produce an error when the " +
			"byte slice is smaller than the public key size.")
	}
}

// Happy path.
func TestTransmitMessage_SetPubKey_GetPubKey_GetPubKeySize(t *testing.T) {
	grp := getGroup()
	pubKey := grp.NewInt(5)
	pubKeySize := 10
	m := newTransmitMessage(255, pubKeySize)

	m.SetPubKey(pubKey)
	testPubKey := m.GetPubKey(grp)

	if pubKey.Cmp(testPubKey) != 0 {
		t.Errorf("GetPubKey() failed to get correct public key."+
			"\nexpected: %s\nreceived: %s", pubKey.Text(10), testPubKey.Text(10))
	}

	if pubKeySize != m.GetPubKeySize() {
		t.Errorf("GetPubKeySize() failed to return the correct size."+
			"\nexpected: %d\nreceived: %d", pubKeySize, m.GetPubKeySize())
	}
}

func TestTransmitMessage_SetPayload_GetPayload_GetPayloadSize(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	pubKeySize := prng.Intn(externalPayloadSize)
	payload := make([]byte, externalPayloadSize-pubKeySize)
	prng.Read(payload)
	m := newTransmitMessage(externalPayloadSize, pubKeySize)

	m.SetPayload(payload)
	testPayload := m.GetPayload()

	if !bytes.Equal(payload, testPayload) {
		t.Errorf("GetPayload() returned incorrect payload."+
			"\nexpected: %+v\nreceived: %+v", payload, testPayload)
	}

	payloadSize := externalPayloadSize - pubKeySize
	if payloadSize != m.GetPayloadSize() {
		t.Errorf("GetPayloadSize() returned incorrect payload size."+
			"\nexpected: %d\nreceived: %d", payloadSize, m.GetPayloadSize())
	}
}

// Error path: supplied payload is not the same size as message payload.
func TestTransmitMessage_SetPayload_PayloadSizeError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetPayload() did not panic when the size of supplied " +
				"payload is not the same size as message payload.")
		}
	}()

	m := newTransmitMessage(255, 10)
	m.SetPayload([]byte{5})
}

// Happy path.
func Test_newTransmitMessagePayload(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	expected := transmitMessagePayload{
		data:     make([]byte, payloadSize),
		rid:      make([]byte, id.ArrIDLen),
		num:      make([]byte, numSize),
		contents: make([]byte, payloadSize-id.ArrIDLen-numSize),
	}

	mp := newTransmitMessagePayload(payloadSize)

	if !reflect.DeepEqual(expected, mp) {
		t.Errorf("newTransmitMessagePayload() did not produce the expected "+
			"transmitMessagePayload.\nexpected: %#v\nreceived: %#v", expected, mp)
	}
}

// Error path: payload size is smaller than than rid size + num size.
func Test_newTransmitMessagePayload_PayloadSizeError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("newTransmitMessagePayload() did not panic when the size " +
				"of the payload is smaller than the size of the reception ID " +
				"+ the message count.")
		}
	}()

	_ = newTransmitMessagePayload(10)
}

// Happy path.
func Test_mapTransmitMessagePayload(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	rid := id.NewIdFromUInt(prng.Uint64(), id.User, t)
	num := uint8(prng.Uint64())
	contents := make([]byte, prng.Intn(1000))
	prng.Read(contents)
	var data []byte
	data = append(data, rid.Bytes()...)
	data = append(data, num)
	data = append(data, contents...)
	mp := mapTransmitMessagePayload(data)

	if !bytes.Equal(data, mp.data) {
		t.Errorf("mapTransmitMessagePayload() failed to map the correct bytes "+
			"for data.\nexpected: %+v\nreceived: %+v", data, mp.data)
	}

	if !bytes.Equal(rid.Bytes(), mp.rid) {
		t.Errorf("mapTransmitMessagePayload() failed to map the correct bytes "+
			"for rid.\nexpected: %+v\nreceived: %+v", rid.Bytes(), mp.rid)
	}

	if num != mp.num[0] {
		t.Errorf("mapTransmitMessagePayload() failed to map the correct bytes "+
			"for num.\nexpected: %d\nreceived: %d", num, mp.num[0])
	}

	if !bytes.Equal(contents, mp.contents) {
		t.Errorf("mapTransmitMessagePayload() failed to map the correct bytes "+
			"for contents.\nexpected: %+v\nreceived: %+v", contents, mp.contents)
	}
}

// Happy path.
func TestTransmitMessagePayload_Marshal_Unmarshal(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	data := make([]byte, prng.Intn(1000))
	prng.Read(data)
	mp := mapTransmitMessagePayload(data)

	payloadBytes := mp.Marshal()

	testPayload, err := unmarshalTransmitMessagePayload(payloadBytes)
	if err != nil {
		t.Errorf("unmarshalTransmitMessagePayload() produced an error: %+v", err)
	}

	if !reflect.DeepEqual(mp, testPayload) {
		t.Errorf("Failed to marshal and unmarshal payload."+
			"\nexpected: %+v\nreceived: %+v", mp, testPayload)
	}
}

// Error path: supplied byte slice is too small for the ID and message count.
func Test_unmarshalTransmitMessagePayload(t *testing.T) {
	_, err := unmarshalTransmitMessagePayload([]byte{6})
	if err == nil {
		t.Error("unmarshalTransmitMessagePayload() did not return an error " +
			"when the supplied byte slice was too small.")
	}
}

// Happy path.
func TestTransmitMessagePayload_SetRID_GetRID(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	mp := newTransmitMessagePayload(prng.Intn(2000))
	rid := id.NewIdFromUInt(prng.Uint64(), id.User, t)

	mp.SetRID(rid)
	testRID := mp.GetRID()

	if !rid.Cmp(testRID) {
		t.Errorf("GetRID() did not return the expected ID."+
			"\nexpected: %s\nreceived: %s", rid, testRID)
	}
}

// Happy path.
func TestTransmitMessagePayload_SetCount_GetCount(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	mp := newTransmitMessagePayload(prng.Intn(2000))
	count := uint8(prng.Uint64())

	mp.SetCount(count)
	testCount := mp.GetCount()

	if count != testCount {
		t.Errorf("GetCount() did not return the expected count."+
			"\nexpected: %d\nreceived: %d", count, testCount)
	}
}

// Happy path.
func TestTransmitMessagePayload_SetContents_GetContents_GetContentsSize(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	mp := newTransmitMessagePayload(payloadSize)
	contentsSize := payloadSize - id.ArrIDLen - numSize
	contents := make([]byte, contentsSize)
	prng.Read(contents)

	mp.SetContents(contents)
	testContents := mp.GetContents()
	if !bytes.Equal(contents, testContents) {
		t.Errorf("GetContents() did not return the expected contents."+
			"\nexpected: %+v\nreceived: %+v", contents, testContents)
	}

	if contentsSize != mp.GetContentsSize() {
		t.Errorf("GetContentsSize() did not return the expected size."+
			"\nexpected: %d\nreceived: %d", contentsSize, mp.GetContentsSize())
	}
}

// Error path: supplied bytes are smaller than payload contents.
func TestTransmitMessagePayload_SetContents(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetContents() did not panic when the size of the " +
				"supplied bytes is not the same as the payload content size.")
		}
	}()

	mp := newTransmitMessagePayload(255)
	mp.SetContents([]byte{1, 2, 3})
}

func getGroup() *cyclic.Group {
	return cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D4941"+
			"3394C049B7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688"+
			"B55B3DD2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861"+
			"575E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC"+
			"718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FF"+
			"B1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBC"+
			"A23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD"+
			"161C7738F32BF29A841698978825B4111B4BC3E1E198455095958333D776D8B2B"+
			"EEED3A1A1A221A6E37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C"+
			"4F50D7D7803D2D4F278DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F"+
			"1390B5D3FEACAF1696015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F"+
			"96789C38E89D796138E6319BE62E35D87B1048CA28BE389B575E994DCA7554715"+
			"84A09EC723742DC35873847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
}
