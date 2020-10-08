package contact

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/bind"
)

type factList struct {
	source *Contact
}

func (fl factList) Num() int {
	return len(fl.source.Facts)
}

func (fl factList) Get(i int) bind.Fact {
	return fl.source.Facts[i]
}

func (fl factList) Add(fact string, factType int) error {
	ft := FactType(factType)
	if !ft.IsValid() {
		return errors.New("Invalid fact type")
	}
	fl.source.Facts = append(fl.source.Facts, Fact{
		Fact: fact,
		T:    ft,
	})
	return nil
}
