package attempts

import (
	"sort"
	"sync"
	"sync/atomic"
)

const maxHistogramSize = 100
const minElements = 3
const percentileNumerator = 66
const percentileDenominator = 99
const percentileDenominatorOffset = 49

type SendAttemptTracker interface {
	SubmitProbeAttempt(numAttemptsUntilSuccessful int)
	GetOptimalNumAttempts() (attempts int, ready bool)
}

type sendAttempts struct {
	lock         sync.Mutex
	numAttempts  []int
	currentIndex int
	isFull       bool

	optimalAttempts *int32
}

func NewSendAttempts() SendAttemptTracker {
	optimalAttempts := int32(-1)

	sa := &sendAttempts{
		numAttempts:     make([]int, maxHistogramSize),
		currentIndex:    0,
		isFull:          false,
		optimalAttempts: &optimalAttempts,
	}
	return sa
}

func (sa *sendAttempts) SubmitProbeAttempt(a int) {
	sa.lock.Lock()
	defer sa.lock.Unlock()

	sa.numAttempts[sa.currentIndex] = a
	sa.currentIndex += 1
	if sa.currentIndex == len(sa.numAttempts) {
		sa.currentIndex = 0
		sa.isFull = true
	}

	sa.computeOptimalUnsafe()
}

func (sa *sendAttempts) GetOptimalNumAttempts() (attempts int, ready bool) {
	optimalAttempts := atomic.LoadInt32(sa.optimalAttempts)

	if optimalAttempts == -1 {
		return 0, false
	}

	return int(optimalAttempts), true
}

func (sa *sendAttempts) computeOptimalUnsafe() {
	toCopy := maxHistogramSize
	if !sa.isFull {
		if sa.currentIndex < minElements {
			return
		}
		toCopy = sa.currentIndex

	}

	histoCopy := make([]int, toCopy)
	copy(histoCopy, sa.numAttempts[:toCopy])

	sort.Slice(histoCopy, func(i, j int) bool {
		return histoCopy[i] < histoCopy[j]
	})

	optimal := histoCopy[((toCopy*percentileNumerator)+percentileDenominatorOffset)/percentileDenominator]
	atomic.StoreInt32(sa.optimalAttempts, int32(optimal))
}
