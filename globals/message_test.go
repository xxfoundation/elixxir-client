package globals

import (
	//"encoding/binary"
	//"fmt"
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

	shortSlc := []byte(testStrings[0])

	tmpslc := make([]byte, PAYLOAD_LEN-uint32(len(shortSlc)))

	expectedSlices[0] = append(expectedSlices[0], tmpslc...)
	expectedSlices[0] = append(expectedSlices[0], shortSlc...)

	expectedSlices[1] = ([]byte(testStrings[1]))[0:PAYLOAD_LEN]

	expectedSlices[2] = ([]byte(testStrings[2]))[0:PAYLOAD_LEN]

	for i := 0; i < tests; i++ {
		msg := NewMessage(uint64(i), testStrings[i])

		if uint64(i) != msg.senderID {
			t.Errorf("Test of NewMessage failed on test %v, sID did not "+
				"match;\n  Expected: %v, Received: %v", i, i, msg.senderID)
		}

		pl := msg.payload[:]

		if !compareByteSlices(&pl, &expectedSlices[i]) {
			t.Errorf("Test of NewMessage failed on test %v, bytes did not "+
				"match;\n Len Expected: %v, Len Received: %v", i, len(pl),
				len(expectedSlices[i]))
		}

	}

}

func TestConstructDeconstructMessageBytes(t *testing.T) {
	testString := "the game"

	msg := NewMessage(uint64(42), testString)

	dmsg := msg.DeconstructMessageToBytes()

	rtnmsg := *(ConstructMessageFromBytes(dmsg))

	if rtnmsg.senderID != msg.senderID {
		t.Errorf("Test of Message Construction/Deconstruction failed, sID did"+
			" not match;\n  Expected: %v, Received: %v", msg.senderID, rtnmsg.senderID)
	}
	rpl := rtnmsg.payload[:]
	dpl := msg.payload[:]

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

func TestGenerateReceptipientIDBytes(t *testing.T) {
	rid := uint64(2)

	ridbytes := GenerateReceptipientIDBytes(rid)

	if len(*ridbytes) != 512 {
		t.Errorf("Test of GenerateReceptipientIDBytes failed, Incorrect "+
			"Length;\n Expected: %v, Received: %v", 512, len(*ridbytes))
	}

	if (*ridbytes)[511] != byte(rid) {
		t.Errorf("Test of GenerateReceptipientIDBytes failed, Incorrect "+
			"rid;\n Expected: %v, Received: %v", byte(rid), (*ridbytes)[511])
	}
}

//TODO: Test End cases, messages over 2x length, at max length, and others.
