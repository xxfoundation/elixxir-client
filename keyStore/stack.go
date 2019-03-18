package keyStore

// Stack based on last in, first out.
type LIFO []interface{}

// Creates a new stack.
func NewLIFO() *LIFO {
	var lifo LIFO
	lifo = make([]interface{}, 0)
	return &lifo
}

// Returns the length of the stack.
func (lifo *LIFO) Len() uint {
	return uint(len(*lifo))
}

// Adds the specified element to the end of the stack.
func (lifo *LIFO) Push(i interface{}) {
	*lifo = append(*lifo, i)
}

// Returns and removes the last element to be put on the stack.
func (lifo *LIFO) Pop() interface{} {
	var rtn interface{}

	if lifo.Len() == 0 {
		rtn = nil
	} else {
		rtn = (*lifo)[len(*lifo)-1]
		*lifo = (*lifo)[:len(*lifo)-1]
	}

	return rtn
}

// Returns the element at the specified index in the stack or nil if the index
// is out of range. The element is not removed from the stack.
func (lifo *LIFO) Peak(i uint) interface{} {
	if i >= lifo.Len() {
		return nil
	} else {
		return (*lifo)[i]
	}
}

// Returns and removes the element at the specified index in the stack or
// returns nil if the index is out of range.
func (lifo *LIFO) Remove(i uint) interface{} {
	var rtn interface{}

	if i >= lifo.Len() {
		rtn = nil
	} else if lifo.Len()-1 == i {
		rtn = lifo.Pop()
	} else {
		rtn = (*lifo)[i]
		*lifo = append((*lifo)[:i], (*lifo)[i+1:]...)
	}

	return rtn
}
