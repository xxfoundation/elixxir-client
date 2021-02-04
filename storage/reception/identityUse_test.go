package reception

import (
	"math/rand"
	"testing"
	"time"
)

func TestIdentityUse_SetSamplingPeriod(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	const numTests = 1000

	for i := 0; i < numTests; i++ {
		// Generate an identity use
		start := randate()
		end := start.Add(time.Duration(rand.Uint64() % uint64(92*time.Hour)))
		mask := time.Duration(rand.Uint64() % uint64(92*time.Hour))
		iu := IdentityUse{
			Identity: Identity{
				StartValid:  start,
				EndValid:    end,
				RequestMask: mask,
			},
		}

		// Generate the sampling period
		var err error
		iu, err = iu.setSamplingPeriod(rng)
		if err != nil {
			t.Errorf("Errored in generatign sampling "+
				"period on interation %v: %+v", i, err)
		}

		// Test that the range between the periods is correct
		resultRange := iu.EndRequest.Sub(iu.StartRequest)
		expectedRange := iu.EndValid.Sub(iu.StartValid) + iu.RequestMask

		if resultRange != expectedRange {
			t.Errorf("The generated sampling period is of the wrong "+
				"size: Expecterd: %s, Received: %s", expectedRange, resultRange)
		}

		// Test the sampling range does not exceed a reasonable lower bound
		lowerBound := iu.StartValid.Add(-iu.RequestMask)
		if !iu.StartRequest.After(lowerBound) {
			t.Errorf("Start request exceeds the reasonable lower "+
				"bound: \n\t Bound: %s\n\t Start: %s", lowerBound, iu.StartValid)
		}

		// Test the sampling range does not exceed a reasonable upper bound
		upperBound := iu.EndValid.Add(iu.RequestMask - time.Millisecond)
		if iu.EndRequest.After(upperBound) {
			t.Errorf("End request exceeds the reasonable upper bound")
		}
	}

}

func randate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}
