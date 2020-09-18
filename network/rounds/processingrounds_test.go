package rounds

// Testing functions for Processing Round structure

import (
	"gitlab.com/elixxir/client/vendor/gitlab.com/xx_network/primitives/id"
	"testing"
)

// Test that the Processing function inserts the round properly
func TestProcessingRounds_Add(t *testing.T) {
	pr := processing{rounds: make(map[id.Round]struct{})}
	pr.Add(id.Round(10))
	if _, ok := pr.rounds[10]; !ok {
		t.Errorf("Could not find round 10 after it was inserted into the map")
	}
}

// Test that the IsProcessing function correctly finds the round
func TestProcessingRounds_IsProcessing(t *testing.T) {
	pr := processing{rounds: make(map[id.Round]struct{})}
	pr.rounds[id.Round(10)] = struct{}{}
	if !pr.IsProcessing(id.Round(10)) {
		t.Errorf("IsProcessing reported round 10 is not processing after being set as processing")
	}
}

// Test that the Done function removes the processing round
func TestProcessingRounds_Remove(t *testing.T) {
	pr := processing{rounds: make(map[id.Round]struct{})}
	pr.rounds[id.Round(10)] = struct{}{}
	pr.Remove(id.Round(10))
	if _, ok := pr.rounds[id.Round(10)]; ok {
		t.Errorf("Round 10 was not removed from processing list when calling Done")
	}
}
