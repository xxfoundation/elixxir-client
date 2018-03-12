////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"testing"
)

func TestNewMessage(t *testing.T) {

	tests := 3

	testStrings := make([]string, tests)

	testStrings[0] = "short test"
	testStrings[1] = "Perfect test: Lorem ipsum dolor sit amet, consectetur " +
		"adipiscing elit. Curabitur congue, tellus non rhoncus tincidunt, " +
		"tortor mi rhoncus arcu, quis commodo diam elit nec nisl. Phasellus " +
		"luctus velit a tempus rutrum. Etiam sollicitudin a lorem eget " +
		"consequat. Nunc volutpat diam a vulputate blandit. Fusce congue " +
		"laoreet dignissim. Curabitur fermentum lacus vel mauris mollis, in " +
		"tempor ligula ornare. Ut sit amet arcu tellus. Aenean luctus massa " +
		"lorem, id tempus odio faucibus quis. Curabitur cras amet."

	testStrings[2] = "long test: Lorem ipsum dolor sit amet, consectetur " +
		"adipiscing elit. Quisque vitae elit venenatis, tincidunt tellus " +
		"non, efficitur eros. Maecenas vel fermentum magna, ac varius velit." +
		"Mauris eleifend ullamcorper velit, at aliquam magna semper cursus." +
		"Mauris finibus mauris in suscipit placerat. Mauris fermentum dolor " +
		"nisi, a condimentum lacus imperdiet at. Interdum et malesuada fames " +
		"ac ante ipsum primis in faucibus. Mauris hendrerit nisi in suscipit " +
		"ornare. Maecenas imperdiet luctus tincidunt. Vivamus tortor turpis, " +
		"aliquam facilisis bibendum a, efficitur lobortis dolor. Etiam " +
		"iaculis nunc nec convallis condimentum. Vivamus et mauris vel " +
		"sapien efficitur elementum. Vestibulum ante ipsum primis in " +
		"faucibus orci luctus et ultrices posuere cubilia Curae;  " +
		"Ut fermentum aliquet ornare. Sed tincidunt interdum est sed " +
		"vestibulum. Integer ultricies vitae magna ac venenatis. Curabitur " +
		"a velit sit amet erat tincidunt ullamcorper a id nulla. " +
		"Pellentesque habitant morbi tristique senectus et netus et cras " +
		"amet."

	expectedSlices := make([][]byte, tests)

	expectedSlices[0] = []byte(testStrings[0])

	expectedSlices[1] = ([]byte(testStrings[1]))[0:PAYLOAD_LEN]

	expectedSlices[2] = ([]byte(testStrings[2]))[0:PAYLOAD_LEN]

	for i := 0; i < tests; i++ {
		msg := NewMessage(uint64(i+1), uint64(i+1), testStrings[i])[0]

		if uint64(i+1) != msg.senderID.Uint64() {
			t.Errorf("Test of NewMessage failed on test %v, sID did not "+
				"match;\n  Expected: %v, Received: %v", i, i, msg.senderID)
		}

		if uint64(i+1) != msg.recipientID.Uint64() {
			t.Errorf("Test of NewMessage failed on test %v, rID did not "+
				"match;\n  Expected: %v, Received: %v", i, i, msg.recipientID)
		}

		expct := cyclic.NewIntFromBytes(expectedSlices[i])

		if msg.payload.Cmp(expct) != 0 {
			t.Errorf("Test of NewMessage failed on test %v, bytes did not "+
				"match;\n Value Expected: %v, Value Received: %v", i,
				string(expct.Bytes()), string(msg.payload.Bytes()))
		}

	}

}

func TestConstructDeconstructMessageBytes(t *testing.T) {
	testString := "the game"

	msg := NewMessage(uint64(42), uint64(69), testString)[0]

	msg.recipientInitVect.SetInt64(1)

	dmsg := msg.ConstructMessageBytes()

	rtnmsg := dmsg.DeconstructMessageBytes()

	if rtnmsg.senderID.Cmp(msg.senderID) != 0 {
		t.Errorf("Test of Message Construction/Deconstruction failed, sID did"+
			" not match;\n  Expected: %v, Received: %v", msg.senderID.Text(10),
			rtnmsg.senderID.Text(10))
	}
	rpl := rtnmsg.payload.Bytes()[:]
	dpl := msg.payload.Bytes()[:]

	if !compareByteSlices(&rpl, &dpl) {
		t.Errorf("Test of Message Construction/Deconstruction failed, payloads did" +
			" not match;")
	}

}

func compareByteSlices(a, b *[]byte) bool {
	if len(*a) != len(*b) {
		return false
	}

	for i := 0; i < len(*a); i++ {
		if (*a)[i] != (*b)[i] {

			return false
		}

	}

	return true
}

//TODO: Test End cases, messages over 2x length, at max length, and others.
