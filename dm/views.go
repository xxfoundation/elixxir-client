////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import "time"

type ModelConversation struct {
	Pubkey         []byte `json:"pub_key"`
	Nickname       string `json:"nickname"`
	Token          uint32 `json:"token"`
	CodesetVersion uint8  `json:"codeset_version"`

	// Deprecated: KV is the source of truth for blocked users.
	BlockedTimestamp *time.Time `json:"blocked_timestamp"`
}
