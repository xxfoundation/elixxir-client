////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"fmt"
	"gitlab.com/xx_network/primitives/id"
	"math"
	"strconv"
	"time"
)

const (
	defaultPrintInterval   = math.MaxInt64
	debugHeader            = "---------------------------%s----------------------------" + lineEnd
	hostPoolHeader         = "Host-Pool Information"
	hostPoolTableHeader    = "Node ID            | Position" + lineEnd
	removedNodeTableHeader = "Node ID            | Time of Removal" + lineEnd
	lineEnd                = "\r\n"
	nodeIdLength           = 10
	removedListHeader      = "Removed Host Information"
	abbreviatedIdTrailer   = "..."
)

// GoString is a stringer which will return a tabular format of the hostPool.
// This adheres to fmt.GoStringer.
//
// Example Output:
//
//	 ---------------------------Host-Pool Information----------------------------
//		Node ID            | Position
//		ZHVtbXkAAA...      | 4
//		s3XJj1Bjv4...      | 2
//		xwtYNogeq2...      | 0
func (hp *hostPool) GoString() string {
	p := hp.readPool.Load().(*pool)

	toPrint := fmt.Sprintf(debugHeader, hostPoolHeader)

	// Print out the information from the host pool
	toPrint += fmt.Sprintf(hostPoolTableHeader)
	for nodeId, position := range p.hostMap {
		nodePrint := fmt.Sprintf("%s      | %s %s",
			abbreviateNodeId(nodeId), strconv.Itoa(int(position)), lineEnd)
		toPrint += nodePrint
	}

	return toPrint + lineEnd
}

type removedNodes map[id.ID]time.Time

// GoString is a stringer which will return a tabular format of the removedNodes.
//
// Example Output:
//
//	 ---------------------------Removed Host Information----------------------------
//		Node ID            | Time of Removal
//		dKBA9j9esD...      | 2022-12-30 20:45:03.991557276 +0000 UTC
//		s3XJj1Bjv4...      | 2022-12-30 20:47:01.011790913 +0000 UTC
func (rm *removedNodes) GoString() string {
	toPrint := fmt.Sprintf(debugHeader, removedListHeader)

	// Print out the information from the removed nodes list
	toPrint += fmt.Sprintf(removedNodeTableHeader)
	for removedNode, removalTimestamp := range *rm {
		removedPrint := fmt.Sprintf("%s      | %s %s",
			abbreviateNodeId(removedNode), removalTimestamp.UTC().String(), lineEnd)
		toPrint += removedPrint
	}

	return toPrint + lineEnd
}

// abbreviateNodeId is a helper function which outputs an abbreviated node
// id.ID. This will abbreviate it by providing the first nodeIdLength characters
// followed by an abbreviatedIdTrailer. For example, the ID
// "lYbnSItLHwbOyst/WMp18J6gsWasSGXiGBjF+UiO13EB" would be abbreviated to
// "lYbnSItLHw...".
func abbreviateNodeId(nodeId id.ID) string {
	return nodeId.String()[:nodeIdLength] + abbreviatedIdTrailer
}
