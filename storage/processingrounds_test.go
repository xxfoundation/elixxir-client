package storage

// Testing functions for Processing Round structure

import (
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Test that the Processing function inserts the round properly
func TestProcessingRounds_Processing(t *testing.T) {
	pr := ProcessingRounds{rounds: make(map[id.Round]struct{})}
	pr.Processing(id.Round(10))
	if _, ok := pr.rounds[10]; !ok {
		t.Errorf("Could not find round 10 after it was inserted into the map")
	}
}

// Test that the IsProcessing function correctly finds the round
func TestProcessingRounds_IsProcessing(t *testing.T) {
	pr := ProcessingRounds{rounds: make(map[id.Round]struct{})}
	pr.rounds[id.Round(10)] = struct{}{}
	if !pr.IsProcessing(id.Round(10)) {
		t.Errorf("IsProcessing reported round 10 is not processing after being set as processing")
	}
}

// Test that the Done function removes the processing round
func TestProcessingRounds_Done(t *testing.T) {
	pr := ProcessingRounds{rounds: make(map[id.Round]struct{})}
	pr.rounds[id.Round(10)] = struct{}{}
	pr.Done(id.Round(10))
	if _, ok := pr.rounds[id.Round(10)]; ok {
		t.Errorf("Round 10 was not removed from processing list when calling Done")
	}
}
