////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/xx_network/comms/connect"
	"sync"
	"time"
)

func (hp *hostPool) nodeTester(stop *stoppable.Single) {

	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case queryList := <-hp.testNodes:
			jww.DEBUG.Printf("[NodeTester] Received queryList of nodes to test: %v", queryList)
			//test all nodes, find the best
			resultList := make([]time.Duration, len(queryList))

			wg := sync.WaitGroup{}
			for i := 0; i < len(queryList); i++ {
				wg.Add(1)
				go func(hostToQuery *connect.Host, index int) {
					latency, pinged := hostToQuery.IsOnline()
					if !pinged {
						latency = 0
					}
					resultList[index] = latency
					wg.Done()
				}(queryList[i], i)
			}

			//wait until all tests complete
			wg.Wait()

			//find the fastest one which is not 0 (designated as failure)
			lowestLatency := time.Hour
			var bestHost *connect.Host
			for i := 0; i < len(queryList); i++ {
				if resultList[i] != 0 && resultList[i] < lowestLatency {
					lowestLatency = resultList[i]
					bestHost = queryList[i]
				}
			}

			if bestHost != nil {
				//connect to the host then send it over to be
				//added to the host pool

				err := bestHost.Connect()

				if err == nil {
					// Check remote certificates for web hosts
					err = hp.cc.CheckRemoteCertificate(bestHost)
					if err == nil {
						select {
						case hp.newHost <- bestHost:
						default:
							jww.ERROR.Printf("failed to send best host to main thread, " +
								"will be dropped, new addRequest to be sent")
							bestHost = nil
						}
					} else {
						jww.WARN.Printf("Remote certificate check for bestHost %s failed with error %+v, will be dropped", bestHost.GetId(), err)
						bestHost = nil
					}

				} else {
					jww.WARN.Printf("Failed to connect to bestHost %s with error %+v, will be dropped", bestHost.GetId(), err)
					bestHost = nil
				}

			}

			// send the tested nodes back to be labeled as available again
			select {
			case hp.doneTesting <- queryList:
				jww.DEBUG.Printf("[NodeTester] Completed testing query list %s", queryList)
			default:
				jww.ERROR.Printf("failed to send queryList to main thread, " +
					"nodes are stuck in testing, this should never happen")
				bestHost = nil
			}

			if bestHost == nil {
				jww.WARN.Printf("No host selected, restarting the request process")
				// if none of the hosts could be contacted, send a signal
				// to Add a new node to the pool
				select {
				case hp.addRequest <- nil:
				default:
					jww.WARN.Printf("Failed to send a signal to Add hosts after " +
						"testing failure")
				}
			}

		}

	}

}
