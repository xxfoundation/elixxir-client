///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partner

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Subtest: unmarshal/marshal with one session in the buff
func TestRelationship_MarshalUnmarshal(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	// Serialization should include session slice only
	serialized, err := sb.marshal()
	if err != nil {
		t.Fatal(err)
	}

	sb2 := &relationship{
		grp:         mgr.grp,
		t:           0,
		kv:          sb.kv,
		sessions:    make([]*session.Session, 0),
		sessionByID: make(map[session.SessionID]*session.Session),
	}

	err = sb2.unmarshal(serialized)
	if err != nil {
		t.Fatal(err)
	}

	// compare sb2 session list and map
	if !relationshipsEqual(sb, sb2) {
		t.Error("session buffs not equal")
	}
}

// Shows that Relationship returns an equivalent session buff to the one that was saved
func TestLoadRelationship(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	err := sb.save()
	if err != nil {
		t.Fatal(err)
	}

	sb2, err := LoadRelationship(kv, session.Send, mgr.myID, mgr.partner, mockCyHandler{}, mgr.grp, mgr.rng)
	if err != nil {
		t.Fatal(err)
	}

	if !relationshipsEqual(sb, sb2) {
		t.Error("session buffers not equal")
	}
}

// Shows that a deleted Relationship can no longer be pulled from store
func TestDeleteRelationship(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)

	// Generate send relationship
	mgr.send = NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)
	if err := mgr.send.save(); err != nil {
		t.Fatal(err)
	}

	// Generate receive relationship
	mgr.receive = NewRelationship(kv, session.Receive, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)
	if err := mgr.receive.save(); err != nil {
		t.Fatal(err)
	}

	err := DeleteRelationship(mgr)
	if err != nil {
		t.Fatalf("DeleteRelationship error: Could not delete manager: %v", err)
	}

	_, err = LoadRelationship(kv, session.Send, mgr.myID, mgr.partner, mockCyHandler{}, mgr.grp, mgr.rng)
	if err == nil {
		t.Fatalf("DeleteRelationship error: Should not have loaded deleted relationship: %v", err)
	}

	_, err = LoadRelationship(kv, session.Receive, mgr.myID, mgr.partner, mockCyHandler{}, mgr.grp, mgr.rng)
	if err == nil {
		t.Fatalf("DeleteRelationship error: Should not have loaded deleted relationship: %v", err)
	}
}

// Shows that a deleted relationship fingerprint can no longer be pulled from store
func TestRelationship_deleteRelationshipFingerprint(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("deleteRelationshipFingerprint error: " +
				"Did not panic when loading deleted fingerprint")
		}
	}()

	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	err := sb.save()
	if err != nil {
		t.Fatal(err)
	}

	err = deleteRelationshipFingerprint(mgr.kv)
	if err != nil {
		t.Fatalf("deleteRelationshipFingerprint error: "+
			"Could not delete fingerprint: %v", err)
	}

	loadRelationshipFingerprint(mgr.kv)
}

// Shows that Relationship returns a valid session buff
func TestNewRelationshipBuff(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	if sb.sessionByID == nil || len(sb.sessionByID) != 1 {
		t.Error("session map should not be nil, and should have one " +
			"element")
	}
	if sb.sessions == nil || len(sb.sessions) != 1 {
		t.Error("session list should not be nil, and should have one " +
			"element")
	}
}

// Shows that AddSession adds one session to the relationship
func TestRelationship_AddSession(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	if len(sb.sessions) != 1 {
		t.Error("starting session slice length should be 1")
	}
	if len(sb.sessionByID) != 1 {
		t.Error("starting session map length should be 1")
	}

	grp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	partnerID := id.NewIdFromString("zezima", id.User, t)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)
	frng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	s := session.NewSession(kv, session.Send, partnerID,
		myPrivKey, partnerPubKey, baseKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, []byte(""), session.Sending,
		session.GetDefaultParams(), mockCyHandler{}, grp, frng)
	// Note: AddSession doesn't change the session relationship or set anything else up
	// to match the session to the session buffer. To work properly, the session
	// should have been created using the same relationship (which is not the case in
	// this test.)
	sb.AddSession(myPrivKey, partnerPubKey, baseKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	if len(sb.sessions) != 2 {
		t.Error("ending session slice length should be 2")
	}
	if len(sb.sessionByID) != 2 {
		t.Error("ending session map length should be 2")
	}
	if s.GetID() != sb.sessions[0].GetID() {
		t.Error("session added should have same ID")
	}
}

// GetNewest should get the session that was most recently added to the buff
func TestRelationship_GetNewest(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	// The newest session should be nil upon session buffer creation
	nilSession := sb.GetNewest()
	if nilSession == nil {
		t.Error("should not have gotten a nil session from a buffer " +
			"with one session")
	}

	grp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	partnerID := id.NewIdFromString("zezima", id.User, t)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)
	frng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	s := session.NewSession(kv, session.Send, partnerID,
		myPrivKey, partnerPubKey, baseKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, []byte(""), session.Unconfirmed,
		session.GetDefaultParams(), mockCyHandler{}, grp, frng)

	sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	if s.GetID() != sb.GetNewest().GetID() {
		t.Error("session added should have same ID")
	}

	session2 := session.NewSession(kv, session.Send, partnerID,
		myPrivKey, partnerPubKey, baseKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, []byte(""), session.Unconfirmed,
		session.GetDefaultParams(), mockCyHandler{}, grp, frng)
	sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	if session2.GetID() != sb.GetNewest().GetID() {
		t.Error("session added should have same ID")
	}

}

// Shows that Confirm confirms the specified session in the buff
func TestRelationship_Confirm(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	grp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)

	sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	sb.sessions[0].SetNegotiationStatus(session.Sent)

	if sb.sessions[0].IsConfirmed() {
		t.Error("session should not be confirmed before confirmation")
	}

	err := sb.Confirm(sb.sessions[0].GetID())
	if err != nil {
		t.Fatal(err)
	}

	if !sb.sessions[0].IsConfirmed() {
		t.Error("session should be confirmed after confirmation")
	}
}

// Shows that the session buff returns an error when the session doesn't exist
func TestRelationship_Confirm_Err(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	grp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	partnerID := id.NewIdFromString("zezima", id.User, t)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)
	frng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	s := session.NewSession(kv, session.Send, partnerID,
		myPrivKey, partnerPubKey, baseKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, []byte(""), session.Unconfirmed,
		session.GetDefaultParams(), mockCyHandler{}, grp, frng)

	err := sb.Confirm(s.GetID())
	if err == nil {
		t.Error("Confirming a session not in the buff should result in an error")
	}
}

// Shows that a session can get got by ID from the buff
func TestRelationship_GetByID(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	grp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)

	s := sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	session2 := sb.GetByID(s.GetID())
	if !reflect.DeepEqual(s, session2) {
		t.Error("gotten session should be the same")
	}
}

// Shows that GetNewestRekeyableSession acts as expected:
// returning sessions that are confirmed and past rekeyThreshold
func TestRelationship_GetNewestRekeyableSession(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)
	baseKey := session.GenerateE2ESessionBaseKey(mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.grp,
		mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)
	sb.AddSession(mgr.originMyPrivKey, mgr.originPartnerPubKey, baseKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, sid, session.Unconfirmed, session.GetDefaultParams())
	// no available rekeyable sessions: nil
	session2 := sb.getNewestRekeyableSession()
	if session2 != sb.sessions[1] {
		t.Error("newest rekeyable session should be the unconfired session")
	}

	_ = sb.AddSession(mgr.originMyPrivKey, mgr.originPartnerPubKey, baseKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, sid, session.Sending, session.GetDefaultParams())
	sb.sessions[0].SetNegotiationStatus(session.Confirmed)
	session3 := sb.getNewestRekeyableSession()

	if session3 == nil {
		t.Error("no session returned")
	} else if session3.GetID() != sb.sessions[0].GetID() {
		t.Error("didn't get the expected session")
	}

	// add another rekeyable session: that session
	// show the newest session is selected
	_ = sb.AddSession(mgr.originMyPrivKey, mgr.originPartnerPubKey, baseKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, sid, session.Sending, session.GetDefaultParams())

	sb.sessions[0].SetNegotiationStatus(session.Confirmed)

	session4 := sb.getNewestRekeyableSession()
	if session4 == nil {
		t.Error("no session returned")
	} else if session4.GetID() != sb.sessions[0].GetID() {
		t.Error("didn't get the expected session")
	}

	// make the very newest session unrekeyable: the previous session
	// sb.sessions[1].negotiationStatus = Confirmed
	// sb.sessions[0].SetNegotiationStatus(session.Unconfirmed) TODO: do we want a setter here?

	session5 := sb.getNewestRekeyableSession()
	if session5 == nil {
		t.Error("no session returned")
	} else if session5.GetID() != sb.sessions[1].GetID() {
		t.Error("didn't get the expected session")
	}
}

// Shows that GetSessionForSending follows the hierarchy of sessions correctly
func TestRelationship_GetSessionForSending(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	sb.sessions = make([]*session.Session, 0)
	sb.sessionByID = make(map[session.SessionID]*session.Session)

	none := sb.getSessionForSending()
	if none != nil {
		t.Error("getSessionForSending should return nil if there aren't any sendable sessions")
	}

	grp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)

	s, kv := session.CreateTestSession(2000, 1000, 1000, session.Unconfirmed, t)
	_ = sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	sb.sessions[0] = s
	sending := sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}
	if sending.Status() != session.RekeyNeeded || sending.IsConfirmed() {
		t.Errorf("returned session is expected to be 'RekeyNedded' "+
			"'Unconfirmed', it is: %s, confirmed: %v", sending.Status(),
			sending.IsConfirmed())
	}

	s2, _ := session.CreateTestSession(2000, 2000, 1000, session.Unconfirmed, t)
	_ = sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	sb.sessions[0] = s2
	sending = sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}

	if sending.Status() != session.Active || sending.IsConfirmed() {
		t.Errorf("returned session is expected to be 'Active' "+
			"'Unconfirmed', it is: %s, confirmed: %v", sending.Status(),
			sending.IsConfirmed())
	}

	// Third case: confirmed rekey
	s3, _ := session.CreateTestSession(2000, 600, 1000, session.Confirmed, t)
	_ = sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	sb.sessions[0] = s3
	sending = sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}
	if sending.Status() != session.RekeyNeeded || !sending.IsConfirmed() {
		t.Errorf("returned session is expected to be 'RekeyNeeded' "+
			"'Confirmed', it is: %s, confirmed: %v", sending.Status(),
			sending.IsConfirmed())
	}

	// Fourth case: confirmed active
	s4, _ := session.CreateTestSession(2000, 2000, 1000, session.Confirmed, t)
	_ = sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())

	sb.sessions[0] = s4

	sending = sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Errorf("got an unexpected session of state: %s", sending.Status())
	}
	if sending.Status() != session.Active || !sending.IsConfirmed() {
		t.Errorf("returned session is expected to be 'Active' "+
			"'Confirmed', it is: %s, confirmed: %v", sending.Status(),
			sending.IsConfirmed())
	}
}

// Shows that GetKeyForRekey returns a key if there's an appropriate session for rekeying
func TestSessionBuff_GetKeyForRekey(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	sb.sessions = make([]*session.Session, 0)
	sb.sessionByID = make(map[session.SessionID]*session.Session)

	// no available rekeyable sessions: error
	key, err := sb.getKeyForRekey()
	if err == nil {
		t.Error("should have returned an error with no sessions available")
	}
	if key != nil {
		t.Error("shouldn't have returned a key with no sessions available")
	}

	grp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)

	_ = sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	sb.sessions[0].SetNegotiationStatus(session.Confirmed)
	key, err = sb.getKeyForRekey()
	if err != nil {
		t.Error(err)
	}
	if key == nil {
		t.Error("should have returned a valid key with a rekeyable session available")
	}
}

// Shows that GetKeyForSending returns a key if there's an appropriate session for sending
func TestSessionBuff_GetKeyForSending(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner, mgr.originMyPrivKey, mgr.originPartnerPubKey, mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey, session.GetDefaultParams(), mockCyHandler{}, mgr.grp, mgr.rng)

	sb.sessions = make([]*session.Session, 0)
	sb.sessionByID = make(map[session.SessionID]*session.Session)

	// no available rekeyable sessions: error
	key, err := sb.getKeyForSending()
	if err == nil {
		t.Error("should have returned an error with no sessions available")
	}
	if key != nil {
		t.Error("shouldn't have returned a key with no sessions available")
	}

	grp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)

	_ = sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	key, err = sb.getKeyForSending()
	if err != nil {
		t.Error(err)
	}
	if key == nil {
		t.Error("should have returned a valid key with a sendable session available")
	}
}

// Shows that TriggerNegotiation sets up for negotiation correctly
func TestSessionBuff_TriggerNegotiation(t *testing.T) {
	mgr, kv := makeTestRelationshipManager(t)
	sb := NewRelationship(kv, session.Send, mgr.myID, mgr.partner,
		mgr.originMyPrivKey, mgr.originPartnerPubKey,
		mgr.originMySIDHPrivKey, mgr.originPartnerSIDHPubKey,
		session.GetDefaultParams(), mockCyHandler{},
		mgr.grp, mgr.rng)

	sb.sessions = make([]*session.Session, 0)
	sb.sessionByID = make(map[session.SessionID]*session.Session)

	grp := getGroup()
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	sid := session.GetSessionIDFromBaseKey(baseKey)

	session1 := sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	session1.SetNegotiationStatus(session.Confirmed)
	// The added session isn't ready for rekey, so it's not returned here
	negotiations := sb.TriggerNegotiation()
	if len(negotiations) != 0 {
		t.Errorf("should have had zero negotiations: %+v", negotiations)
	}

	// Make only a few keys available to trigger the rekeyThreshold
	session2, _ := session.CreateTestSession(0, 4, 0, session.Sending, t)
	_ = sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, session.GetDefaultParams())
	sb.sessions[0] = session2
	session2.SetNegotiationStatus(session.Confirmed)
	negotiations = sb.TriggerNegotiation()
	if len(negotiations) != 1 {
		t.Fatal("should have had one negotiation")
	}
	if negotiations[0].GetID() != session2.GetID() {
		t.Error("negotiated sessions should include the rekeyable " +
			"session")
	}
	if session2.NegotiationStatus() != session.NewSessionTriggered {
		t.Errorf("Trigger negotiations should have set status to "+
			"triggered: %s", session2.NegotiationStatus())
	}

	// Unconfirmed sessions should also be included in the list
	// as the client should attempt to confirm them
	p := session.GetDefaultParams()
	//set the retry ratio so the unconfirmed session is always retried
	p.UnconfirmedRetryRatio = 1
	session3 := sb.AddSession(myPrivKey, partnerPubKey, nil,
		mySIDHPrivKey, partnerSIDHPubKey,
		sid, session.Sending, p)
	session3.SetNegotiationStatus(session.Unconfirmed)

	// Set session 2 status back to Confirmed to show that more than one session can be returned
	session2.SetNegotiationStatus(session.Confirmed)
	// Trigger negotiations
	negotiations = sb.TriggerNegotiation()

	if len(negotiations) != 2 {
		t.Errorf("num of negotiated sessions here should be 2, have %d", len(negotiations))
	}
	found := false
	for i := range negotiations {
		if negotiations[i].GetID() == session3.GetID() {
			found = true
			if negotiations[i].NegotiationStatus() != session.Sending {
				t.Error("triggering negotiation should change session3 to sending")
			}
		}
	}
	if !found {
		t.Error("session3 not found")
	}

	found = false
	for i := range negotiations {
		if negotiations[i].GetID() == session2.GetID() {
			found = true
		}
	}
	if !found {
		t.Error("session2 not found")
	}
}

func makeTestRelationshipManager(t *testing.T) (*Manager, *versioned.KV) {
	grp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myID := id.NewIdFromString("zezima", id.User, t)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	kv := versioned.NewKV(make(ekv.Memstore))
	frng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	return &Manager{
		kv:                      kv,
		myID:                    myID,
		partner:                 id.NewIdFromString("zezima", id.User, t),
		originMyPrivKey:         myPrivKey,
		originPartnerPubKey:     partnerPubKey,
		originMySIDHPrivKey:     mySIDHPrivKey,
		originPartnerSIDHPubKey: partnerSIDHPubKey,
		grp:                     grp,
		rng:                     frng,
	}, kv
}

func Test_relationship_getNewestRekeyableSession(t *testing.T) {
	// TODO: Add test cases.
}
