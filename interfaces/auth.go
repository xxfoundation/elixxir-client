////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package interfaces

import (
	"gitlab.com/elixxir/crypto/contact"
)

type RequestCallback func(requestor contact.Contact)
