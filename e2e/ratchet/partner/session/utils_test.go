package session

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
)

// Make a default test session with some things populated
func makeTestSession() (*Session, *versioned.KV) {
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

	baseKey := GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	kv := versioned.NewKV(make(ekv.Memstore))
	sid := GetSessionIDFromBaseKey(baseKey)

	s := &Session{
		baseKey:           baseKey,
		myPrivKey:         myPrivKey,
		partnerPubKey:     partnerPubKey,
		mySIDHPrivKey:     mySIDHPrivKey,
		partnerSIDHPubKey: partnerSIDHPubKey,
		e2eParams:         GetDefaultE2ESessionParams(),
		sID:               sid,
		kv:                kv.Prefix(MakeSessionPrefix(sid)),
		t:                 Receive,
		negotiationStatus: Confirmed,
		rekeyThreshold:    5,
		partner:           &id.ID{},
		grp:               grp,
		cyHandler:         mockCyHandler{},
		rng:               fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
	}
	var err error
	s.keyState, err = util.NewStateVector(s.kv,
		"", 1024)
	if err != nil {
		panic(err)
	}
	return s, kv
}

func getGroup() *cyclic.Group {
	e2eGrp := cyclic.NewGroup(
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

	return e2eGrp

}

func getSession(t *testing.T) *Session {
	if t == nil {
		panic("getSession is a testing function and should be called from a test")
	}
	grp := getGroup()
	rng := csprng.NewSystemRNG()

	// finalizeKeyNegotation the baseKey and session
	privateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	publicKey := dh.GeneratePublicKey(privateKey, grp)

	// SIDH keys
	pubA := sidh.NewPublicKey(sidh.Fp434, sidh.KeyVariantSidhA)
	privA := sidh.NewPrivateKey(sidh.Fp434, sidh.KeyVariantSidhA)
	privA.Generate(rng)
	privA.GeneratePublicKey(pubA)
	pubB := sidh.NewPublicKey(sidh.Fp434, sidh.KeyVariantSidhB)
	privB := sidh.NewPrivateKey(sidh.Fp434, sidh.KeyVariantSidhB)
	privB.Generate(rng)
	privB.GeneratePublicKey(pubB)

	baseKey := GenerateE2ESessionBaseKey(privateKey, publicKey, grp, privA,
		pubB)

	keyState, err := util.NewStateVector(versioned.NewKV(make(ekv.Memstore)), "keyState", rand.Uint32())
	if err != nil {
		panic(err)
	}

	return &Session{
		baseKey:  baseKey,
		keyState: keyState,
	}
}

func getFingerprint() *format.Fingerprint {
	rand.Seed(netTime.Now().UnixNano())
	fp := format.Fingerprint{}
	rand.Read(fp[:])

	return &fp
}

// compare fields also represented in SessionDisk
// fields not represented in SessionDisk shouldn't be expected to be populated by Unmarshal
func cmpSerializedFields(a *Session, b *Session) error {
	if a.negotiationStatus != b.negotiationStatus {
		return errors.New("confirmed differed")
	}
	if a.t != b.t {
		return errors.New("t differed")
	}
	if a.e2eParams.MaxKeys != b.e2eParams.MaxKeys {
		return errors.New("maxKeys differed")
	}
	if a.e2eParams.MinKeys != b.e2eParams.MinKeys {
		return errors.New("minKeys differed")
	}
	if a.e2eParams.NumRekeys != b.e2eParams.NumRekeys {
		return errors.New("NumRekeys differed")
	}
	if a.baseKey.Cmp(b.baseKey) != 0 {
		return errors.New("baseKey differed")
	}
	if a.myPrivKey.Cmp(b.myPrivKey) != 0 {
		return errors.New("myPrivKey differed")
	}
	if a.partnerPubKey.Cmp(b.partnerPubKey) != 0 {
		return errors.New("partnerPubKey differed")
	}
	return nil
}

type mockCyHandler struct {
}

func (m mockCyHandler) AddKey(k *Cypher) {
	return
}

func (m mockCyHandler) DeleteKey(k *Cypher) {
	return
}
