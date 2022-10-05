package xxdk

import (
	"github.com/beevik/ntp"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/xx_network/primitives/netTime"
	"sync/atomic"
	"time"
)

const (
	numOffsets = 5
	runDelta   = 5 * 60 * time.Second
)

var host = "0.north-america.pool.ntp.org"

type NtpTime struct {
	offsets       []time.Duration
	averageOffset *int64
}

func InitNTP() *NtpTime {
	averageOffset := int64(0)
	nt := &NtpTime{
		offsets:       make([]time.Duration, 0, numOffsets),
		averageOffset: &averageOffset,
	}
	netTime.Now = nt.Now
	return nt
}

func (nt *NtpTime) Start() (stoppable.Stoppable, error) {
	stopper := stoppable.NewSingle("ntp")
	go func() {
		for true {
			offset := nt.sync(host)
			if len(nt.offsets) == numOffsets {
				jww.INFO.Printf("Updated ntp time with '%s' current offset "+
					"of %s", host, time.Duration(offset))
				select {
				case <-stopper.Quit():
					return
				case <-time.After(runDelta):
				}
			}

		}
	}()
	return stopper, nil
}

func (nt *NtpTime) Now() time.Time {
	localNow := time.Now()
	offset := time.Duration(atomic.LoadInt64(nt.averageOffset))
	return localNow.Add(offset)
}

func (nt *NtpTime) sync(host string) int64 {

	response, _ := ntp.Query(host)

	if len(nt.offsets) >= numOffsets {
		nt.offsets = append(nt.offsets[1:], response.ClockOffset)
	} else {
		nt.offsets = append(nt.offsets, response.ClockOffset)
	}

	offsetSum := int64(0)
	for _, offset := range nt.offsets {
		offsetSum += int64(offset)
	}

	offsetAvg := (offsetSum + int64(len(nt.offsets))/2) / int64(len(nt.offsets))

	atomic.StoreInt64(nt.averageOffset, offsetAvg)
	return offsetAvg
}
