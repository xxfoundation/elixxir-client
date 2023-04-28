package sync

import "encoding/json"

type set map[string]struct{}

func newSet(size uint) set {
	if size == 0 {
		return make(set)
	} else {
		return make(set, size)
	}
}

func (ks set) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &set{})
}

func (ks set) MarshalJSON() ([]byte, error) {
	return json.Marshal(&ks)
}

func (ks set) Has(element string) bool {
	_, ok := ks[element]
	return ok
}

func (ks set) Add(element string) {
	ks[element] = struct{}{}
}

func (ks set) Delete(element string) {
	delete(ks, element)
}
