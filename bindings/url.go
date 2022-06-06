///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"fmt"
	"gitlab.com/xx_network/primitives/id"
)

const dashboardBaseURL = "https://dashboard.xx.network"

func getRoundURL(round id.Round) string {
	return fmt.Sprintf("%s/rounds/%d?xxmessenger=true", dashboardBaseURL, round)
}
