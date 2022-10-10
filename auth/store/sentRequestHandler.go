////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

// SentRequestHandler allows the lower level to assign and remove services
type SentRequestHandler interface {
	Add(sr *SentRequest)
	Delete(sr *SentRequest)
}
