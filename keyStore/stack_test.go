package keyStore

import (
	"math"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

var randVals []int
var size int

func TestMain(m *testing.M) {
	// Generate slice of random integers
	size = 10
	randVals = make([]int, size)
	rand.Seed(42)
	for i := 0; i < size; i++ {
		randVals[i] = rand.Intn(math.MaxUint32)
	}

	os.Exit(m.Run())
}

// Tests that NewLIFO() creates a new LIFO stack with a length of zero.
func TestNewLIFO(t *testing.T) {
	lifo := NewLIFO()

	if len(*lifo) != 0 {
		t.Errorf("NewLIFO() created new stack with incorrect length\n\treceived: %v\n\texpected: %v", len(*lifo), 0)
	}
}

// Tests that Len() returns the correct length on an empty stack.
func TestLIFO_Len_EmptyStack(t *testing.T) {
	lifo := NewLIFO()

	if lifo.Len() != 0 {
		t.Errorf("Len() did not return zero on an empty stack\n\treceived: %v\n\texpected: %v", lifo.Len(), 0)
	}
}

// Tests that Len() returns the correct length on a filled stack.
func TestLIFO_Len_FilledStack(t *testing.T) {
	lifo := NewLIFO()

	// Add random integers into the stack
	for i := 0; i < size; i++ {
		*lifo = append(*lifo, randVals[i])
	}

	if lifo.Len() != uint(size) {
		t.Errorf("Len() did not return the correct length on a filled stack\n\treceived: %v\n\texpected: %v", lifo.Len(), uint(size))
	}
}

// Tests that Push() places elements on the stack in the correct order.
func TestLIFO_Push(t *testing.T) {
	lifo := NewLIFO()

	// Push random integers onto the stack
	for i := 0; i < size; i++ {
		lifo.Push(randVals[i])
	}

	// Test to make sure all items are on the stack
	for i := 0; i < size; i++ {
		if (*lifo)[i] != randVals[i] {
			t.Errorf("Push() did not put the correct value onto the stack\n\treceived: %v\n\texpected: %v", (*lifo)[i], randVals[i])
		}
	}
}

// Tests that Pop() returns nil on an empty stack.
func TestLIFO_Pop_EmptyStack(t *testing.T) {
	lifo := NewLIFO()

	if lifo.Pop() != nil {
		t.Errorf("Pop() did not return nil on an empty stack\n\treceived: %v\n\texpected: %v", lifo.Pop(), nil)
	}
}

// Tests that Pop() removes the correct values from the stack in the correct
// order.
func TestLIFO_Pop_FilledStack(t *testing.T) {
	lifo := NewLIFO()

	// Push random integers onto stack
	for i := 0; i < size; i++ {
		lifo.Push(randVals[i])
	}

	// Test to make sure all items are Popped off the stack
	for i := size - 1; i >= 0; i-- {
		if lifo.Pop() != randVals[i] {
			t.Errorf("Pop() did not remove the correct value from the stack\n\treceived: %v\n\texpected: %v", lifo.Pop(), randVals[i])
		}
	}
}

// Tests that Peak() returns nil on an empty stack.
func TestLIFO_Peak_EmptyStack(t *testing.T) {
	lifo := NewLIFO()

	if lifo.Peak(0) != nil {
		t.Errorf("Peak() did not return nil on an empty stack\n\treceived: %v\n\texpected: %v", lifo.Peak(0), nil)
	}
}

// Tests that Peak() returns the correct value at the specified index.
func TestLIFO_Peak_FilledStack(t *testing.T) {
	lifo := NewLIFO()

	// Push random integers onto the stack
	for i := 0; i < size; i++ {
		lifo.Push(randVals[i])
	}

	// Test to make sure all items are on the stack
	for i := 0; i < size; i++ {
		if lifo.Peak(uint(i)) != randVals[i] {
			t.Errorf("Peak() did not return the correct value from the stack at index %v\n\treceived: %v\n\texpected: %v", i, lifo.Peak(uint(i)), randVals[i])
		}
	}
}

// Tests that Peak() returns nil for an index out of range.
func TestLIFO_Peak_IndexOutOfRange(t *testing.T) {
	lifo := NewLIFO()

	// Push random integers onto the stack
	for i := 0; i < size; i++ {
		lifo.Push(randVals[i])
	}

	if lifo.Peak(uint(size+5)) != nil {
		t.Errorf("Peak() did not return nil for an index out of range\n\treceived: %v\n\texpected: %v", lifo.Peak(uint(size+5)), nil)
	}
}

// Tests that Remove() returns nil on an empty stack.
func TestLIFO_Remove_EmptyStack(t *testing.T) {
	lifo := NewLIFO()

	if lifo.Remove(1) != nil {
		t.Errorf("Remove() did not return nil on an empty stack\n\treceived: %v\n\texpected: %v", lifo.Remove(0), nil)
	}
}

// Tests that Remove() deletes the correct elements from a filled stack.
func TestLIFO_Remove_FilledStack(t *testing.T) {
	lifo := NewLIFO()
	lifo2 := NewLIFO()

	// Push random integers onto the stack
	for i := 0; i < size; i++ {
		lifo.Push(randVals[i])

		// Assemble final stack to compare to
		if i%2 == 0 {
			lifo2.Push(randVals[i])
		}
	}

	// Remove every other element from stack and make sure it is correct
	for i := size - 1; i >= 0; i -= 2 {
		if lifo.Remove(uint(i)) != randVals[i] {
			t.Errorf("Remove() did not remove the correct value from the stack at index %v\n\treceived: %v\n\texpected: %v", i, lifo.Remove(uint(i)), randVals[i])
		}
	}

	// Compare resulting stack to expected final stack
	if !reflect.DeepEqual(lifo, lifo2) {
		t.Errorf("Remove() did not leave the resulting list correct\n\treceived: %v\n\texpected: %v", lifo, lifo2)
	}
}

// Tests that Remove() returns nil for an index out of range.
func TestLIFO_Remove_IndexOutOfRange(t *testing.T) {
	lifo := NewLIFO()

	// Push random integers onto the stack
	for i := 0; i < size; i++ {
		lifo.Push(randVals[i])
	}

	if lifo.Remove(uint(size+5)) != nil {
		t.Errorf("Remove() did not return nil for an index out of range\n\treceived: %v\n\texpected: %v", lifo.Remove(uint(size+5)), nil)
	}
}
