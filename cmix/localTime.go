////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import "time"

// localTime describes a local time object, which gets time from the local clock
// in milliseconds.
type localTime struct{}

// NowMs returns the current time in milliseconds.
func (localTime) NowMs() int64 {
	t := time.Now()
	return (t.UnixNano() + int64(time.Millisecond)/2 + 1) / int64(time.Millisecond)
}
