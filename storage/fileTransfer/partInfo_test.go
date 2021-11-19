////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////
package fileTransfer

import (
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"reflect"
	"testing"
)

// Tests that newPartInfo creates the expected partInfo.
func Test_newPartInfo(t *testing.T) {
	// Created expected partInfo
	expected := &partInfo{
		id:    ftCrypto.UnmarshalTransferID([]byte("TestTransferID")),
		fpNum: 16,
	}

	// Create new partInfo
	received := newPartInfo(expected.id, expected.fpNum)

	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("New partInfo does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that a partInfo that is marshalled with partInfo.marshal and
// unmarshalled with unmarshalPartInfo matches the original.
func Test_partInfo_marshal_unmarshalPartInfo(t *testing.T) {
	expectedPI := newPartInfo(
		ftCrypto.UnmarshalTransferID([]byte("TestTransferID")), 25)

	piBytes := expectedPI.marshal()
	receivedPI := unmarshalPartInfo(piBytes)

	if !reflect.DeepEqual(expectedPI, receivedPI) {
		t.Errorf("Marshalled and unmarshalled partInfo does not match original."+
			"\nexpected: %+v\nreceived: %+v", expectedPI, receivedPI)
	}
}

// Consistency test of partInfo.String.
func Test_partInfo_String(t *testing.T) {
	prng := NewPrng(42)
	expectedStrings := []string{
		"{id:U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVI= fpNum:0}",
		"{id:39ebTXZCm2F6DJ+fDTulWwzA1hRMiIU1hBrL4HCbB1g= fpNum:1}",
		"{id:CD9h03W8ArQd9PkZKeGP2p5vguVOdI6B555LvW/jTNw= fpNum:2}",
		"{id:uoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44= fpNum:3}",
		"{id:GwuvrogbgqdREIpC7TyQPKpDRlp4YgYWl4rtDOPGxPM= fpNum:4}",
		"{id:rnvD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHA= fpNum:5}",
		"{id:ceeWotwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGE= fpNum:6}",
		"{id:SYlH/fNEQQ7UwRYCP6jjV2tv7Sf/iXS6wMr9mtBWkrE= fpNum:7}",
		"{id:NhnnOJZN/ceejVNDc2Yc/WbXT+weG4lJGrcjbkt1IWI= fpNum:8}",
		"{id:kM8r60LDyicyhWDxqsBnzqbov0bUqytGgEAsX7KCDog= fpNum:9}",
		"{id:XTJg8d6XgoPUoJo2+WwglBdG4+1NpkaprotPp7T8OiA= fpNum:10}",
		"{id:uvoade0yeoa4sMOa8c/Ss7USGep5Uzq/RI0sR50yYHU= fpNum:11}",
	}

	for i, expected := range expectedStrings {
		tid, _ := ftCrypto.NewTransferID(prng)
		pi := newPartInfo(tid, uint16(i))

		if expected != pi.String() {
			t.Errorf("partInfo #%d string does not match expected."+
				"\nexpected: %s\nreceived: %s", i, expected, pi.String())
		}
	}
}
