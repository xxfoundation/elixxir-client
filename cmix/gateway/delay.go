package gateway

import (
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"time"
)

// piecewise table of delay to percent of the bucket that
// is full
var table = map[float64]time.Duration{
	0:   0,
	0.1: 0,
	0.2: 0,
	0.3: 0,
	0.4: 100 * time.Millisecond,
	0.5: 500 * time.Millisecond,
	0.6: 2 * time.Second,
	0.7: 5 * time.Second,
	0.8: 8 * time.Second,
	0.9: 9 * time.Second,
	1.0: 10 * time.Second,
	1.1: 10 * time.Second,
}

// getDelay computes the delay through linear
func getDelay(bucket float64, poolsize uint) time.Duration {
	ratio := bucket / float64(poolsize)

	if ratio < 0 {
		ratio = 0
	}

	if ratio > 1 {
		ratio = 1
	}

	ratioFloor := math.Floor(ratio)
	ratioCeil := math.Ceil(ratio)

	upperRatio := ratio - ratioFloor
	lowerRatio := 1 - upperRatio

	bottom := time.Duration(float64(table[ratioFloor]) * lowerRatio)
	top := time.Duration(float64(table[ratioCeil]) * upperRatio)

	return top + bottom
}

// bucket is a leaky bucket implementation.
type bucket struct {
	//time until the entire bucket is leaked
	leakRate time.Duration
	points   float64
	lastEdit time.Time
	poolsize uint
}

// newBucket initializes a new bucket.
func newBucket(poolSize int) *bucket {
	return &bucket{
		leakRate: time.Duration(poolSize) * table[1],
		points:   0,
		lastEdit: netTime.Now(),
		poolsize: uint(poolSize),
	}
}

func (b *bucket) leak() {
	now := netTime.Now()

	delta := now.Sub(b.lastEdit)
	if delta < 0 {
		return
	}

	leaked := (float64(delta) / float64(b.leakRate)) * float64(b.poolsize)
	b.points -= leaked
	if b.points < 0 {
		b.points = 0
	}
}

func (b *bucket) Add() {
	b.leak()
	b.points += 1
}

func (b *bucket) Reset() {
	b.points = 0
}

func (b *bucket) GetDelay() time.Duration {
	b.leak()
	return getDelay(b.points, b.poolsize)
}
