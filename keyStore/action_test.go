package keyStore

import "testing"

// Test all outputs of String for coverage
func TestAction_String(t *testing.T) {
	expectedStr := "None"
	action := None
	if action.String() != expectedStr {
		t.Errorf("String returned %s, expected %s",
			action.String(),
			expectedStr)
	}

	expectedStr = "Rekey"
	action = Rekey
	if action.String() != expectedStr {
		t.Errorf("String returned %s, expected %s",
			action.String(),
			expectedStr)
	}

	expectedStr = "Purge"
	action = Purge
	if action.String() != expectedStr {
		t.Errorf("String returned %s, expected %s",
			action.String(),
			expectedStr)
	}

	expectedStr = "Deleted"
	action = Deleted
	if action.String() != expectedStr {
		t.Errorf("String returned %s, expected %s",
			action.String(),
			expectedStr)
	}
}
