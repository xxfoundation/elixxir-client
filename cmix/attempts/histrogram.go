package attempts

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	maxHistogramSize            = 100
	minElements                 = 3
	percentileNumerator         = 66
	percentileDenominator       = 99
	percentileDenominatorOffset = 49
	optimalAttemptsInitValue    = -1
)

// SendAttemptTracker tracks the number of attempts it took to send a cMix
// message in order to predict how many attempt are needed.
type SendAttemptTracker interface {
	// SubmitProbeAttempt feeds the number of attempts it took to send a cMix
	// message into the tracker and updates the optimal number of attempts.
	SubmitProbeAttempt(numAttemptsUntilSuccessful int)

	// GetOptimalNumAttempts returns the number of optimal sends. If there is
	// insufficient data to calculate, then ready is false.
	GetOptimalNumAttempts() (attempts int, ready bool)
}

// sendAttempts tracks the number of attempts to send a cMix message.
type sendAttempts struct {
	optimalAttempts *int32
	isFull          bool
	currentIndex    int
	numAttempts     []int
	lock            sync.Mutex
}

// NewSendAttempts initialises a new SendAttemptTracker.
func NewSendAttempts() SendAttemptTracker {
	optimalAttempts := int32(optimalAttemptsInitValue)
	sa := &sendAttempts{
		optimalAttempts: &optimalAttempts,
		isFull:          false,
		currentIndex:    0,
		numAttempts:     make([]int, maxHistogramSize),
	}

	return sa
}

// SubmitProbeAttempt feeds the number of attempts it took to send a cMix
// message into the tracker and updates the optimal number of attempts.
func (sa *sendAttempts) SubmitProbeAttempt(numAttemptsUntilSuccessful int) {
	sa.lock.Lock()
	defer sa.lock.Unlock()

	sa.numAttempts[sa.currentIndex] = numAttemptsUntilSuccessful
	sa.currentIndex++

	if sa.currentIndex == len(sa.numAttempts) {
		sa.currentIndex = 0
		sa.isFull = true
	}

	sa.computeOptimalUnsafe()
}

// GetOptimalNumAttempts returns the number of optimal sends. If there is
// insufficient data to calculate, then ready is false.
func (sa *sendAttempts) GetOptimalNumAttempts() (attempts int, ready bool) {
	optimalAttempts := atomic.LoadInt32(sa.optimalAttempts)

	if optimalAttempts == optimalAttemptsInitValue {
		return 0, false
	}

	return int(optimalAttempts), true
}

// computeOptimalUnsafe updates the optimal send attempts.
func (sa *sendAttempts) computeOptimalUnsafe() {
	toCopy := maxHistogramSize
	if !sa.isFull {
		if sa.currentIndex < minElements {
			return
		}
		toCopy = sa.currentIndex
	}

	histogramCopy := make([]int, toCopy)
	copy(histogramCopy, sa.numAttempts[:toCopy])
	sort.Ints(histogramCopy)

	i := ((toCopy * percentileNumerator) + percentileDenominatorOffset) /
		percentileDenominator
	optimal := histogramCopy[i]
	atomic.StoreInt32(sa.optimalAttempts, int32(optimal))
}

// String prints the values in the sendAttempts in a human-readable form for
// debugging and logging purposes. This function adheres to the fmt.Stringer
// interface.
func (sa *sendAttempts) String() string {
	fields := []string{
		"optimalAttempts:" + strconv.Itoa(int(atomic.LoadInt32(sa.optimalAttempts))),
		"isFull:" + strconv.FormatBool(sa.isFull),
		"currentIndex:" + strconv.Itoa(sa.currentIndex),
		"numAttempts:" + fmt.Sprintf("%d", sa.numAttempts),
	}

	return "{" + strings.Join(fields, " ") + "}"
}
