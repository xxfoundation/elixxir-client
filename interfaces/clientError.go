////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package interfaces

type ClientError struct {
	Source  string
	Message string
	Trace   string
}

type ClientErrorReport func(source, message, trace string)
