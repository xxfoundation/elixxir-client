package bindings

import (
	"testing"
	"strings"
)

func TestGetContactListJSON(t *testing.T) {
	// This call includes validating the JSON against the schema
	result, err := GetContactListJSON()

	if err != nil {
		t.Error(err.Error())
	}

	// But, just in case,
	// let's make sure that we got the error out of validateContactList anyway
	err = validateContactListJSON(result)

	if err != nil {
		t.Error(err.Error())
	}

	// Finally, make sure that all the names we expect are in the JSON
	expected := []string{"Ben", "Rick", "Jake", "Mario", "Allan", "David",
	"Jim", "Spencer", "Will", "Jono"}

	actual := string(result)

	for _, nick := range(expected) {
		if !strings.Contains(actual, nick) {
			t.Errorf("Error: Expected name %v wasn't in JSON %v", nick, actual)
		}
	}
}
