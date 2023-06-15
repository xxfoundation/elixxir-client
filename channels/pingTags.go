////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"encoding/hex"
	"strings"

	"github.com/pkg/errors"
)

// PingType describes a user ping. It is used to describe more information about
// the ping to the user.
type PingType string

const (
	// GenericPing indicates a generic notification that is not a reply or
	// mention.
	GenericPing PingType = ""

	// ReplyPing indicates that someone replied to this user's message.
	ReplyPing PingType = "usrReply"

	// MentionPing indicates that someone mentioned (tagged) the user in a
	// message.
	MentionPing PingType = "usrMention"
)

// getRankingPingType returns the ping that ranks higher.
func getRankingPingType(a, b PingType) PingType {
	if pingHierarchy[a] > pingHierarchy[b] {
		return a
	}
	return b
}

// pingHierarchy describes the ranking of ping types. If a user receives a ping
// on multiple tags, the highest ranking ping tag is reported to the user.
var pingHierarchy = map[PingType]uint32{
	GenericPing: 0,
	MentionPing: 100,
	ReplyPing:   200,
}

// delimiter between the public key and ping type in a ping tag.
const pingTagDelim = "-"

// makeUserPingTags creates a list of tags to include in a cmix.Service from
// channel pings.
func makeUserPingTags(pings map[PingType][]ed25519.PublicKey) []string {
	if pings == nil || len(pings) == 0 {
		return nil
	}
	var tags []string
	for pt, users := range pings {
		s := make([]string, len(users))
		for i := range users {
			s[i] = makeUserPingTag(users[i], pt)
		}
		tags = append(tags, s...)
	}

	return tags
}

// makeUserPingTag creates a tag from a user's public key and ping type to be
// used in a tag list in a cmix.Service.
func makeUserPingTag(user ed25519.PublicKey, pt PingType) string {
	return hex.EncodeToString(user) + pingTagDelim + string(pt)
}

// pingTypeFromTag gets the PingType from the ping tag.
func pingTypeFromTag(tag string) (PingType, error) {
	s := strings.SplitN(tag, pingTagDelim, 2)
	if len(s) < 2 {
		return "", errors.New("invalid ping tag")
	}

	return PingType(s[1]), nil
}
