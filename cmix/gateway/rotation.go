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
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/crypto/randomness"
	"math/big"
	"time"
)

// Rotation is a long-running thread which will cause the runner thread
// to shuffle the hostPool. This will be done over a random interval, with
// the interval being randomly selected every Rotation. The Rotation
// is performed by sending a nil signal to hostPool.addRequest, which
// will force random updates to the hostPool.
func (hp *hostPool) Rotation(stop *stoppable.Single) {
	for {

		delay := hp.params.RotationPeriod
		if hp.params.RotationPeriodVariability != 0 {
			stream := hp.rng.GetStream()

			seed := make([]byte, 32)
			_, err := stream.Read(seed)
			if err != nil {
				jww.FATAL.Panicf("Failed to read (rng): %+v", err)
			}
			h, _ := hash.NewCMixHash()
			r := randomness.RandInInterval(big.NewInt(int64(hp.params.RotationPeriodVariability)), seed, h).Int64()
			r = r - (r / 2)

			delay = delay + time.Duration(r)
		}

		t := time.NewTimer(delay)

		select {
		case <-stop.Quit():
			stop.ToStopped()
			t.Stop()
			return
		case <-t.C:
			select {
			case hp.addRequest <- nil:
			default:
				jww.WARN.Printf("Failed to send an Add request after %s delay", delay)
			}
		}
	}
}
