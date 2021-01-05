package parse

import (
	"bytes"
	"reflect"
	"testing"
)

// This tests that a new function part is successfully created
func Test_newMessagePart(t *testing.T) {
	gotmp := newMessagePart(32, 6, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'})
	expectedmp := messagePart{
		Data: []uint8{0x0, 0x0, 0x0, 0x20, 0x6, 0x0, 0x7, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67},
		Id:   []uint8{0x0, 0x0, 0x0, 0x20}, Part: []uint8{0x6},
		Len:      []uint8{0x0, 0x7},
		Contents: []uint8{0x74, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67},
	}
	if !reflect.DeepEqual(gotmp, expectedmp) {
		t.Errorf("MessagePart received and MessagePart expected do not match.\n\tGot: %#v\n\tExpected: %#v", gotmp, expectedmp)
	}
}

func TestMessagePart_GetID(t *testing.T) {
	gotmp := newMessagePart(32, 6,
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g'})
	if gotmp.GetID() != 32 {
		t.Errorf("received and expected do not match."+
			"\n\tGot: %#v\n\tExpected: %#v", gotmp.GetID(), 32)
	}
}

func TestMessagePart_GetPart(t *testing.T) {
	gotmp := newMessagePart(32, 6,
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g'})
	if gotmp.GetPart() != 6 {
		t.Errorf("received and expected do not match."+
			"\n\tGot: %#v\n\tExpected: %#v", gotmp.GetPart(), 6)
	}
}

func TestMessagePart_GetContents(t *testing.T) {
	gotmp := newMessagePart(32, 6,
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g'})
	if bytes.Compare(gotmp.GetContents(), []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}) != 0 {
		t.Errorf("received and expected do not match."+
			"\n\tGot: %#v\n\tExpected: %#v", gotmp.GetContents(), 6)
	}
}

func TestMessagePart_GetSizedContents(t *testing.T) {
	gotmp := newMessagePart(32, 6,
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g'})
	if bytes.Compare(gotmp.GetSizedContents(), []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}) != 0 {
		t.Errorf("received and expected do not match."+
			"\n\tGot: %#v\n\tExpected: %#v", gotmp.GetSizedContents(), 6)
	}
}

func TestMessagePart_GetContentsLength(t *testing.T) {
	gotmp := newMessagePart(32, 6,
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g'})
	if gotmp.GetContentsLength() != 7 {
		t.Errorf("received and expected do not match."+
			"\n\tGot: %#v\n\tExpected: %#v", gotmp.GetContentsLength(), 7)
	}
}
