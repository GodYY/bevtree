package bevtree

type nodeStackElem struct {
	node node
	next *nodeStackElem
}

type nodeStack struct {
	topElem *nodeStackElem
}

func (s *nodeStack) empty() bool { return s.topElem == nil }

func (s *nodeStack) push(node node) {
	top := &nodeStackElem{
		node: node,
		next: s.topElem,
	}

	s.topElem = top
}

func (s *nodeStack) pop() node {
	if s.topElem == nil {
		return nil
	}

	top := s.topElem
	s.topElem = s.topElem.next
	top.next = nil
	return top.node
}

func (s *nodeStack) top() node {
	if s.topElem == nil {
		return nil
	}

	return s.topElem.node
}

func (s *nodeStack) clear() {
	s.topElem = nil
}

type nodeQueueElem struct {
	q     *nodeQueue
	node  node
	prev_ *nodeQueueElem
	next_ *nodeQueueElem
}

func (e *nodeQueueElem) next() *nodeQueueElem {
	if p := e.next_; e.q != nil && p != &e.q.root {
		return p
	}
	return nil
}

func (e *nodeQueueElem) prev() *nodeQueueElem {
	if p := e.prev_; e.q != nil && p != &e.q.root {
		return p
	}
	return nil
}

type nodeQueue struct {
	root nodeQueueElem
	len_ int
}

func newNodeQueue() *nodeQueue { return new(nodeQueue).init() }

func (q *nodeQueue) init() *nodeQueue {
	q.root.prev_ = &q.root
	q.root.next_ = &q.root
	q.len_ = 0
	return q
}

func (q *nodeQueue) lazyInit() {
	if q.root.next_ == nil {
		q.init()
	}
}

func (q *nodeQueue) len() int { return q.len_ }

func (q *nodeQueue) empty() bool { return q.len_ == 0 }

func (q *nodeQueue) clear(fs ...func(node)) {
	var f func(node)
	if len(fs) > 0 {
		f = fs[0]
	}

	for e := q.root.next_; e != &q.root; e = e.next_ {
		if e.node != nil {
			f(e.node)
		}
	}

	q.init()
}

func (q *nodeQueue) front() *nodeQueueElem {
	if q.len_ == 0 {
		return nil
	}

	return q.root.next_
}

func (q *nodeQueue) back() *nodeQueueElem {
	if q.len_ == 0 {
		return nil
	}

	return q.root.prev_
}

func (q *nodeQueue) frontNode() node {
	if q.len_ == 0 {
		return nil
	}
	return q.root.next_.node
}

func (q *nodeQueue) backNode() node {
	if q.len_ == 0 {
		return nil
	}
	return q.root.prev_.node
}

func (q *nodeQueue) pushFront(node node) *nodeQueueElem {
	q.lazyInit()
	return q.insertNode(node, &q.root)
}

func (q *nodeQueue) pushBack(node node) *nodeQueueElem {
	q.lazyInit()
	return q.insertNode(node, q.root.prev_)
}

func (q *nodeQueue) insertBefore(node node, e *nodeQueueElem) *nodeQueueElem {
	if e.q != q {
		return nil
	}
	return q.insertNode(node, e.prev_)
}

func (q *nodeQueue) insertAfter(node node, e *nodeQueueElem) *nodeQueueElem {
	if e.q != q {
		return nil
	}
	return q.insertNode(node, e)
}

func (q *nodeQueue) insert(e, at *nodeQueueElem) *nodeQueueElem {
	e.prev_ = at
	e.next_ = at.next_
	e.prev_.next_ = e
	e.next_.prev_ = e
	e.q = q
	q.len_++
	return e
}

func (q *nodeQueue) insertNode(node node, at *nodeQueueElem) *nodeQueueElem {
	return q.insert(&nodeQueueElem{node: node}, at)
}

func (q *nodeQueue) move(e, at *nodeQueueElem) *nodeQueueElem {
	if e == at {
		return e
	}

	e.prev_.next_ = e.next_
	e.next_.prev_ = e.prev_
	e.prev_ = at
	e.next_ = at.next_
	e.prev_.next_ = e
	e.next_.prev_ = e
	return e
}

func (q *nodeQueue) moveToFront(e *nodeQueueElem) {
	if e.q != q || e == q.root.next_ {
		return
	}

	q.move(e, &q.root)
}

func (q *nodeQueue) moveToBack(e *nodeQueueElem) {
	if e.q != q || e == q.root.prev_ {
		return
	}

	q.move(e, q.root.prev_)
}

func (q *nodeQueue) moveBefore(e, mark *nodeQueueElem) {
	if e.q != q || e == mark || mark.q != q {
		return
	}

	q.move(e, mark.prev_)
}

func (q *nodeQueue) moveAfter(e, mark *nodeQueueElem) {
	if e.q != q || e == mark || mark.q != q {
		return
	}

	q.move(e, mark)
}

func (q *nodeQueue) remove(e *nodeQueueElem) {
	e.prev_.next_ = e.next_
	e.next_.prev_ = e.prev_
	e.prev_ = nil
	e.next_ = nil
	e.q = nil
	q.len_--
}

func (q *nodeQueue) popFrontNode() node {
	if q.len_ == 0 {
		return nil
	}

	node := q.root.next_.node
	q.remove(q.root.next_)
	return node
}

func (q *nodeQueue) popBackNode() node {
	if q.len_ == 0 {
		return nil
	}

	node := q.root.prev_.node
	q.remove(q.root.next_)
	return node
}
