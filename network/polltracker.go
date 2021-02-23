package network

import (
	"fmt"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

type pollTracker map[id.ID]map[int64]uint

func newPollTracker()*pollTracker{
	pt := make(pollTracker)
	return &pt
}

//tracks a single poll
func (pt *pollTracker)Track(ephID ephemeral.Id, source *id.ID){
	if _, exists := (*pt)[*source]; !exists{
		(*pt)[*source] = make(map[int64]uint)
		(*pt)[*source][ephID.Int64()] = 0
	}else if _, exists := (*pt)[*source][ephID.Int64()]; !exists{
		(*pt)[*source][ephID.Int64()] = 0
	}else{
		(*pt)[*source][ephID.Int64()] = (*pt)[*source][ephID.Int64()] + 1
	}
}

//reports all resent polls
func (pt *pollTracker)Report()string{
	report := ""
	numReports := uint(0)

	for source := range *pt{
		numSubReports := uint(0)
		subReport := ""
		for ephID, reports := range (*pt)[source]{
			numSubReports += reports
			subReport += fmt.Sprintf("\n\t\tEphID %d polled %d times", ephID, reports)
		}
		subReport = fmt.Sprintf("\n\tID %s polled %d times", &source, numSubReports)
		numReports += numSubReports
	}

	return fmt.Sprintf("\nPolled the network %d times", numReports) + report
}

