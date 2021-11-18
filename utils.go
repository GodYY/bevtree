package bevtree

type nodeStackElem struct {
	node Node
	next *nodeStackElem
}

type nodeStack struct {
	topElem *nodeStackElem
}

func (s *nodeStack) empty() bool { return s.topElem == nil }

func (s *nodeStack) push(node Node) {
	top := &nodeStackElem{
		node: node,
		next: s.topElem,
	}

	s.topElem = top
}

func (s *nodeStack) pop() Node {
	if s.topElem == nil {
		return nil
	}

	top := s.topElem
	s.topElem = s.topElem.next
	top.next = nil
	return top.node
}

func (s *nodeStack) top() Node {
	if s.topElem == nil {
		return nil
	}

	return s.topElem.node
}

func (s *nodeStack) clear() {
	s.topElem = nil
}
