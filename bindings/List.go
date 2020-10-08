package bindings

import "errors"

type IntList struct {
	lst []int
}

func MakeIntList() *IntList {
	return &IntList{lst: make([]int, 0)}
}

func (il *IntList) Add(i int) {
	il.lst = append(il.lst, i)
}

func (il *IntList) Len() int {
	return len(il.lst)
}

func (il *IntList) Get(i int) (int, error) {
	if i < 0 || i >= len(il.lst) {
		return 0, errors.New("invalid index")
	}
	return il.lst[i], nil
}
