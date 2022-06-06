///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/large"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// Happy path.
func TestNewRequest(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	pubKeySize := prng.Intn(externalPayloadSize)
	expected := Request{
		data:    make([]byte, externalPayloadSize),
		version: make([]byte, requestVersionSize),
		pubKey:  make([]byte, pubKeySize),
		payload: make([]byte, externalPayloadSize-pubKeySize-requestVersionSize),
	}

	m := NewRequest(externalPayloadSize, pubKeySize)

	if !reflect.DeepEqual(expected, m) {
		t.Errorf("NewRequest did not produce the expected Request."+
			"\nexpected: %#v\nreceived: %#v", expected, m)
	}
}

// Error path: public key size is larger than external payload size.
func TestNewRequest_PubKeySizeError(t *testing.T) {
	externalPayloadSize, pubKeySize := 5, 10
	expectedErr := fmt.Sprintf(
		errNewReqPayloadSize, externalPayloadSize, pubKeySize)
	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("NewRequest did not panic with the expected error when " +
				"the size of the payload is smaller than the size of the " +
				"public key.")
		}
	}()

	_ = NewRequest(externalPayloadSize, pubKeySize)
}

// Happy path.
func Test_mapRequest(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	pubKeySize := prng.Intn(externalPayloadSize)
	pubKey := make([]byte, pubKeySize)
	prng.Read(pubKey)
	payload := make([]byte, externalPayloadSize-pubKeySize)
	prng.Read(payload)
	version := make([]byte, 1)
	var data []byte
	data = append(data, version...)
	data = append(data, pubKey...)
	data = append(data, payload...)
	m := mapRequest(data, pubKeySize)

	if !bytes.Equal(data, m.data) {
		t.Errorf("mapRequest failed to map the correct bytes for data."+
			"\nexpected: %+v\nreceived: %+v", data, m.data)
	}

	if !bytes.Equal(pubKey, m.pubKey) {
		t.Errorf("mapRequest failed to map the correct bytes for pubKey."+
			"\nexpected: %+v\nreceived: %+v", pubKey, m.pubKey)
	}

	if !bytes.Equal(payload, m.payload) {
		t.Errorf("mapRequest failed to map the correct bytes for payload."+
			"\nexpected: %+v\nreceived: %+v", payload, m.payload)
	}
}

// Happy path.
func TestRequest_Marshal_UnmarshalRequest(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	pubKeySize := prng.Intn(externalPayloadSize)
	data := make([]byte, externalPayloadSize)
	prng.Read(data)
	m := mapRequest(data, pubKeySize)

	msgBytes := m.Marshal()

	newMsg, err := UnmarshalRequest(msgBytes, pubKeySize)
	if err != nil {
		t.Errorf("unmarshalRequest produced an error: %+v", err)
	}

	if !reflect.DeepEqual(m, newMsg) {
		t.Errorf("Failed to marshal/unmarshal message."+
			"\nexpected: %+v\nreceived: %+v", m, newMsg)
	}
}

// Error path: public key size is larger than byte slice.
func Test_UnmarshalRequest_PubKeySizeError(t *testing.T) {
	_, err := UnmarshalRequest([]byte{1, 2, 3}, 5)
	if err == nil {
		t.Error("unmarshalRequest did not produce an error when the " +
			"byte slice is smaller than the public key size.")
	}
}

// Happy path.
func TestRequest_SetPubKey_GetPubKey_GetPubKeySize(t *testing.T) {
	grp := getGroup()
	pubKey := grp.NewInt(5)
	pubKeySize := 10
	m := NewRequest(255, pubKeySize)

	m.SetPubKey(pubKey)
	testPubKey := m.GetPubKey(grp)

	if pubKey.Cmp(testPubKey) != 0 {
		t.Errorf("GetPubKey failed to get correct public key."+
			"\nexpected: %s\nreceived: %s", pubKey.Text(10), testPubKey.Text(10))
	}

	if pubKeySize != m.GetPubKeySize() {
		t.Errorf("GetPubKeySize failed to return the correct size."+
			"\nexpected: %d\nreceived: %d", pubKeySize, m.GetPubKeySize())
	}
}

func TestRequest_SetPayload_GetPayload_GetPayloadSize(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	pubKeySize := prng.Intn(externalPayloadSize)
	payloadSize := externalPayloadSize - pubKeySize - requestVersionSize
	payload := make([]byte, payloadSize)
	prng.Read(payload)
	m := NewRequest(externalPayloadSize, pubKeySize)

	m.SetPayload(payload)
	testPayload := m.GetPayload()

	if !bytes.Equal(payload, testPayload) {
		t.Errorf("GetContents returned incorrect payload."+
			"\nexpected: %+v\nreceived: %+v", payload, testPayload)
	}

	if payloadSize != m.GetPayloadSize() {
		t.Errorf("GetContentsSize returned incorrect content size."+
			"\nexpected: %d\nreceived: %d", payloadSize, m.GetPayloadSize())
	}
}

// Error path: supplied payload is not the same size as message payload.
func TestRequest_SetPayload_PayloadSizeError(t *testing.T) {
	m := NewRequest(255, 10)
	payload := []byte{5}
	expectedErr := fmt.Sprintf(errReqPayloadSize, len(m.payload), len(payload))

	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("SetContents did not panic with the expected error when "+
				"the size of supplied contents is not the same size as "+
				"message contents.\nexpected: %s\nreceived: %+v",
				expectedErr, r)
		}
	}()

	m.SetPayload(payload)
}

// Happy path.
func TestNewRequestPayload(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	expected := RequestPayload{
		data:             make([]byte, payloadSize),
		nonce:            make([]byte, nonceSize),
		numRequestParts:  make([]byte, numRequestPartsSize),
		maxResponseParts: make([]byte, maxResponsePartsSize),
		size:             make([]byte, sizeSize),
		contents:         make([]byte, payloadSize-requestMinSize),
	}
	binary.BigEndian.PutUint16(expected.size, uint16(payloadSize-requestMinSize))
	expected.SetMaxResponseParts(10)
	expected.data = append(expected.nonce,
		append(expected.numRequestParts,
			append(expected.maxResponseParts,
				append(expected.size, expected.contents...)...)...)...)

	payload := make([]byte, payloadSize-requestMinSize)
	mp := NewRequestPayload(payloadSize, payload, 10)

	if !reflect.DeepEqual(expected, mp) {
		t.Errorf("NewRequestPayload did not produce the expected "+
			"RequestPayload.\nexpected: %+v\nreceived: %+v", expected, mp)
	}
}

// Error path: payload size is smaller than rid size + maxResponseParts size.
func TestNewRequestPayload_PayloadSizeError(t *testing.T) {
	payloadSize := 10
	prng := rand.New(rand.NewSource(42))
	payload := make([]byte, payloadSize)
	_, err := prng.Read(payload)
	if err != nil {
		t.Errorf("Failed to read to payload: %+v", err)
	}

	expectedErr := fmt.Sprintf(
		errNewReqPayloadPayloadSize, payloadSize, requestMinSize)

	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("NewRequestPayload did not panic with the expected error "+
				"when the size of the payload is smaller than the size of the "+
				"reception ID + the message count.\nexpected: %s\nreceived: %+v",
				expectedErr, r)
		}
	}()

	_ = NewRequestPayload(10, payload, 5)
}

// Happy path.
func Test_mapRequestPayload(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	nonceBytes := make([]byte, nonceSize)
	numRequestParts := make([]byte, numRequestPartsSize)
	numRequestParts[0] = 1
	maxResponseParts := make([]byte, maxResponsePartsSize)
	size := []byte{uint8(prng.Uint64()), uint8(prng.Uint64())}
	contents := make([]byte, prng.Intn(1000))
	prng.Read(contents)
	var data []byte
	data = append(data, nonceBytes...)
	data = append(data, numRequestParts...)
	data = append(data, maxResponseParts...)
	data = append(data, size...)
	data = append(data, contents...)
	mp := mapRequestPayload(data)

	if !bytes.Equal(data, mp.data) {
		t.Errorf("mapRequestPayload failed to map the correct bytes "+
			"for data.\nexpected: %+v\nreceived: %+v", data, mp.data)
	}

	if !bytes.Equal(nonceBytes, mp.nonce) {
		t.Errorf("mapRequestPayload failed to map the correct bytes "+
			"for the nonce.\nexpected: %s\nreceived: %s", nonceBytes, mp.nonce)
	}

	if !bytes.Equal(numRequestParts, mp.numRequestParts) {
		t.Errorf("mapRequestPayload failed to map the correct bytes "+
			"for the numRequestParts.\nexpected: %s\nreceived: %s", nonceBytes, mp.nonce)
	}

	if !bytes.Equal(maxResponseParts, mp.maxResponseParts) {

	}

	if !bytes.Equal(size, mp.size) {
		t.Errorf("mapRequestPayload failed to map the correct bytes "+
			"for size.\nexpected: %+v\nreceived: %+v", size, mp.size)
	}

	if !bytes.Equal(contents, mp.contents) {
		t.Errorf("mapRequestPayload failed to map the correct bytes "+
			"for contents.\nexpected: %+v\nreceived: %+v", contents, mp.contents)
	}
}

// Happy path.
func TestRequestPayload_Marshal_UnmarshalRequestPayload(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	data := make([]byte, prng.Intn(1000))
	prng.Read(data)
	mp := mapRequestPayload(data)

	payloadBytes := mp.Marshal()

	testPayload, err := UnmarshalRequestPayload(payloadBytes)
	if err != nil {
		t.Errorf("UnmarshalRequestPayload produced an error: %+v", err)
	}

	if !reflect.DeepEqual(mp, testPayload) {
		t.Errorf("Failed to marshal and unmarshal payload."+
			"\nexpected: %+v\nreceived: %+v", mp, testPayload)
	}
}

// Error path: supplied byte slice is too small for the ID and message count.
func Test_UnmarshalRequestPayload(t *testing.T) {
	_, err := UnmarshalRequestPayload([]byte{6})
	if err == nil {
		t.Error("UnmarshalRequestPayload did not return an error " +
			"when the supplied byte slice was too small.")
	}
}

// Happy path.
func TestRequestPayload_GetRecipientID(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	payload := make([]byte, payloadSize-requestMinSize)
	_, err := prng.Read(payload)
	if err != nil {
		t.Errorf("Failed to read to payload: %+v", err)
	}

	mp := NewRequestPayload(payloadSize, payload, 5)
	expectedRID := singleUse.NewRecipientID(getGroup().NewInt(42), mp.Marshal())

	testRID := mp.GetRecipientID(getGroup().NewInt(42))

	if !expectedRID.Cmp(testRID) {
		t.Errorf("GetRecipientID did not return the expected ID."+
			"\nexpected: %s\nreceived: %s", expectedRID, testRID)
	}
}

// Happy path.
func TestRequestPayload_SetNonce_GetNonce(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	payload := make([]byte, payloadSize-requestMinSize)
	_, err := prng.Read(payload)
	if err != nil {
		t.Errorf("Failed to read to payload: %+v", err)
	}

	mp := NewRequestPayload(payloadSize, payload, 5)

	expectedNonce := prng.Uint64()
	expectedNonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(expectedNonceBytes, expectedNonce)
	err = mp.SetNonce(strings.NewReader(string(expectedNonceBytes)))
	if err != nil {
		t.Errorf("SetNonce produced an error: %+v", err)
	}

	if expectedNonce != mp.GetNonce() {
		t.Errorf("GetNonce did not return the expected nonce."+
			"\nexpected: %d\nreceived: %d", expectedNonce, mp.GetNonce())
	}
}

// Error path: RNG return an error.
func TestRequestPayload_SetNonce_RngError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	payload := make([]byte, payloadSize-requestMinSize)
	_, err := prng.Read(payload)
	if err != nil {
		t.Errorf("Failed to read to payload: %+v", err)
	}

	mp := NewRequestPayload(payloadSize, payload, 5)
	err = mp.SetNonce(strings.NewReader(""))

	if !check(err, "failed to generate nonce") {
		t.Errorf("SetNonce did not return an error when nonce generation "+
			"fails: %+v", err)
	}
}

// Happy path.
func TestRequestPayload_SetMaxResponseParts_GetMaxResponseParts(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	payload := make([]byte, payloadSize-requestMinSize)
	_, err := prng.Read(payload)
	if err != nil {
		t.Errorf("Failed to read to payload: %+v", err)
	}

	mp := NewRequestPayload(payloadSize, payload, 5)
	count := uint8(prng.Uint64())

	mp.SetMaxResponseParts(count)
	testCount := mp.GetMaxResponseParts()

	if count != testCount {
		t.Errorf("GetNumRequestParts did not return the expected count."+
			"\nexpected: %d\nreceived: %d", count, testCount)
	}
}

// Happy path.
func TestRequestPayload_GetContents_GetContentsSize_GetMaxContentsSize(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := format.MinimumPrimeSize
	contentsSize := (format.MinimumPrimeSize - requestMinSize) / 2
	contents := make([]byte, contentsSize)
	prng.Read(contents)

	mp := NewRequestPayload(payloadSize, contents, 5)

	testContents := mp.GetContents()
	if !bytes.Equal(contents, testContents) {
		t.Errorf("GetContents did not return the expected contents."+
			"\nexpected: %+v\nreceived: %+v", contents, testContents)
	}

	if contentsSize != mp.GetContentsSize() {
		t.Errorf("GetContentsSize did not return the expected size."+
			"\nexpected: %d\nreceived: %d", contentsSize, mp.GetContentsSize())
	}

	if format.MinimumPrimeSize-requestMinSize != mp.GetMaxContentsSize() {
		t.Errorf("GetMaxResponsePartSize did not return the expected size."+
			"\nexpected: %d\nreceived: %d",
			format.MinimumPrimeSize-requestMinSize, mp.GetMaxContentsSize())
	}
}

// Error path: supplied bytes are smaller than payload contents.
func TestRequestPayload_SetContents(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := format.MinimumPrimeSize
	payload := make([]byte, payloadSize-requestMinSize)
	_, err := prng.Read(payload)
	if err != nil {
		t.Errorf("Failed to read to payload: %+v", err)
	}
	mp := NewRequestPayload(payloadSize, payload, 5)
	contents := make([]byte, format.MinimumPrimeSize+1)
	expectedErr := fmt.Sprintf(
		errReqPayloadContentsSize, len(contents), len(mp.contents))

	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("SetContents did not panic with the expected error when "+
				"the size of the supplied bytes is not the same as the "+
				"payload content size.\nexpected: %s\nreceived: %+v",
				expectedErr, r)
		}
	}()

	mp.SetContents(contents)
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

// check returns true if the error is not nil and contains the substring.
func check(err error, subStr string) bool {
	return err != nil && strings.Contains(err.Error(), subStr)
}
