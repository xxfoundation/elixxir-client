///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

//import (
//	"time"
//)

type NodeKeys struct {
	WorkerPoolSize uint
}

func GetDefaultNodeKeys() NodeKeys {
	return NodeKeys{
		WorkerPoolSize: 10,
	}
}
