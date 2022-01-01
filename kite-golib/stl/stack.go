package stl

// Stack is a type
type Stack struct {
	top  *item
	size int
}

type item struct {
	value interface{} // All types satisfy the empty interface, so we can store anything here.
	next  *item
}

// Len returns the stack's length
func (s *Stack) Len() int {
	return s.size
}

// Push a new element onto the stack
func (s *Stack) Push(value interface{}) {
	s.top = &item{value, s.top}
	s.size++
}

// Pop removes the top element from the stack and return it's value
// If the stack is empty, return nil
func (s *Stack) Pop() (value interface{}) {
	if s.size > 0 {
		value, s.top = s.top.value, s.top.next
		s.size--
		return
	}
	return nil
}
