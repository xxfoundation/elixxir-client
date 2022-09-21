////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"reflect"
	"testing"

	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
)

// Helper function which compares 2 partner.Manager's.
func managersEqualLegacySIDH(expected, received partner.ManagerLegacySIDH, t *testing.T) bool {
	equal := true
	if !reflect.DeepEqual(expected.PartnerId(), received.PartnerId()) {
		t.Errorf("Did not Receive expected Manager.partnerID."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.PartnerId(), received.PartnerId())
		equal = false
	}

	if !reflect.DeepEqual(expected.ConnectionFingerprint(), received.ConnectionFingerprint()) {
		t.Errorf("Did not Receive expected Manager.Receive."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.ConnectionFingerprint(),
			received.ConnectionFingerprint())
		equal = false
	}
	if !reflect.DeepEqual(expected.MyId(), received.MyId()) {
		t.Errorf("Did not Receive expected Manager.myId."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.MyId(), received.PartnerId())
		equal = false
	}

	if !reflect.DeepEqual(expected.MyRootPrivateKey(),
		received.MyRootPrivateKey()) {
		t.Errorf("Did not Receive expected Manager.MyPrivateKey."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.MyRootPrivateKey(), received.MyRootPrivateKey())
		equal = false
	}

	if !reflect.DeepEqual(expected.SendRelationshipFingerprint(),
		received.SendRelationshipFingerprint()) {
		t.Errorf("Did not Receive expected Manager."+
			"SendRelationshipFingerprint."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.SendRelationshipFingerprint(),
			received.SendRelationshipFingerprint())
		equal = false
	}

	return equal
}

// Implements a mock session.CypherHandler.
type mockCyHandlerLegacySIDH struct{}

func (m mockCyHandlerLegacySIDH) AddKey(k session.CypherLegacySIDH) {
	return
}

func (m mockCyHandlerLegacySIDH) DeleteKey(k session.CypherLegacySIDH) {
	return
}
