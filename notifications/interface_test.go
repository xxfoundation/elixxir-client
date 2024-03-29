package notifications

import (
	"gitlab.com/xx_network/primitives/id"
	"math"
	"testing"
)

func TestGroup_DeepCopy(t *testing.T) {
	g := make(Group)

	idA := &id.ID{}
	idA[0] = 1

	idB := &id.ID{}
	idB[0] = 2

	g[*idA] = State{
		Metadata: nil,
		Status:   0,
	}

	g[*idB] = State{
		Metadata: nil,
		Status:   1,
	}

	gCopy := g.DeepCopy()

	//check they are the same
	for key, element := range g {
		s, exists := g[key]
		if !exists {
			t.Errorf("element %s not found", &key)
		}
		if s.Status != element.Status {
			t.Errorf("element %s does not match", &key)
		}
	}

	// check that edits do not propagate
	delete(gCopy, *idA)
	gCopy[*idB] = State{Status: 100}

	if _, exists := g[*idA]; !exists {
		t.Errorf("deletion propogated")
	}

	if g[*idB].Status != 1 {
		t.Errorf("edits propogated")
	}
}

func TestNotificationState_String(t *testing.T) {
	inputs := []NotificationState{Mute, WhenOpen, Push, 15}
	outputs := []string{"Mute", "WhenOpen", "Push", "Unknown notifications state: 15"}

	for idx, ns := range inputs {
		if ns.String() != outputs[idx] {
			t.Errorf("wrong string produced for %d; expected: %s, "+
				"received: %s", ns, ns, outputs[idx])
		}
	}
}

func TestNotificationState_IsValid(t *testing.T) {
	inputs := []NotificationState{Mute, WhenOpen, Push, -100, 14, Mute - 1,
		Push + 1, 1000, math.MaxInt64, math.MinInt64}
	outputs := []bool{true, true, true, false, false, false, false, false,
		false, false}

	for idx, ns := range inputs {
		result := ns.IsValid() == nil
		if outputs[idx] != result {
			t.Errorf("on test %d output did not match expected, "+
				"expected: %t, received: %t", idx, outputs[idx], result)
		}
	}
}
