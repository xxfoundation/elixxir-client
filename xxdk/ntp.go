package xxdk

import (
	"github.com/beevik/ntp"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"sync/atomic"
	"time"
)

const (
	numOffsets = 5
	runDelta   = 5 * 60 * time.Second
)

var hosts = []string{"0.north-america.pool.ntp.org", "time-a-g.nist.gov", "time-b-g.nist.gov",
	"time-c-g.nist.gov", "time-d-g.nist.gov", "time-d-g.nist.gov", "time-e-g.nist.gov",
	"time-e-g.nist.gov", "time-a-wwv.nist.gov", "time-b-wwv.nist.gov", "time-c-wwv.nist.gov",
	"time-d-wwv.nist.gov", "time-d-wwv.nist.gov", "time-e-wwv.nist.gov", "time-e-wwv.nist.gov",
	"time-a-b.nist.gov", "time-b-b.nist.gov", "time-c-b.nist.gov", "time-d-b.nist.gov", "time-d-b.nist.gov",
	"time-e-b.nist.gov", "time-e-b.nist.gov", "time.nist.gov", "utcnist.colorado.edu", "utcnist2.colorado.edu",
	"time1.google.com", "time2.google.com", "time3.google.com", "time4.google.com", "0.pool.ntp.org",
	"1.pool.ntp.org", "2.pool.ntp.org", "3.pool.ntp.org", "time.windows.com", "time.apple.com", "time.euro.apple.com"}

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
			host := hosts[rand.Int()%len(hosts)]
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

	response, err := ntp.Query(host)
	if err != nil {
		oldAvg := atomic.LoadInt64(nt.averageOffset)
		jww.WARN.Printf("skipping poll of ntp, current offset: %s, error : %s",
			time.Duration(oldAvg), err)
		return oldAvg
	}

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
