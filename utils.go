package bevtree

import "log"

func assert(b bool, msg interface{}) {
	if !b {
		log.Panic(msg)
	}
}

func assertF(b bool, f string, args ...interface{}) {
	if !b {
		log.Panicf(f, args...)
	}
}

func assertNil(i interface{}, msg interface{}) {
	assert(i != nil, msg)
}

func assertNilF(i interface{}, f string, args ...interface{}) {
	assertF(i != nil, f, args...)
}

func assertNilArg(arg interface{}, name string) {
	if arg == nil {
		log.Panicf("argument \"%s\" nil", name)
	}
}

func must(i, err error) interface{} {
	if err != nil {
		log.Panic(err)
	}

	return i
}

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

type taskQueueElem struct {
	q    *taskQueue
	task task
	prev *taskQueueElem
	next *taskQueueElem
}

func (e *taskQueueElem) getNext() *taskQueueElem {
	if p := e.next; e.q != nil && p != &e.q.root {
		return p
	}
	return nil
}

func (e *taskQueueElem) getPrev() *taskQueueElem {
	if p := e.prev; e.q != nil && p != &e.q.root {
		return p
	}
	return nil
}

type taskQueue struct {
	root taskQueueElem
	len  int
}

func newTaskQueue() *taskQueue { return new(taskQueue).init() }

func (q *taskQueue) init() *taskQueue {
	q.root.prev = &q.root
	q.root.next = &q.root
	q.len = 0
	return q
}

func (q *taskQueue) lazyInit() {
	if q.root.next == nil {
		q.init()
	}
}

func (q *taskQueue) getLen() int { return q.len }

func (q *taskQueue) empty() bool { return q.len == 0 }

func (q *taskQueue) clear(fs ...func(task)) {
	var f func(task)
	if len(fs) > 0 {
		f = fs[0]
	}

	for e := q.root.next; e != &q.root; e = e.next {
		if e.task != nil && f != nil {
			f(e.task)
		}
	}

	q.init()
}

func (q *taskQueue) front() *taskQueueElem {
	if q.len == 0 {
		return nil
	}

	return q.root.next
}

func (q *taskQueue) back() *taskQueueElem {
	if q.len == 0 {
		return nil
	}

	return q.root.prev
}

func (q *taskQueue) frontTask() task {
	if q.len == 0 {
		return nil
	}
	return q.root.next.task
}

func (q *taskQueue) backTask() task {
	if q.len == 0 {
		return nil
	}
	return q.root.prev.task
}

func (q *taskQueue) pushFront(task task) *taskQueueElem {
	q.lazyInit()
	return q.insertTask(task, &q.root)
}

func (q *taskQueue) pushBack(task task) *taskQueueElem {
	q.lazyInit()
	return q.insertTask(task, q.root.prev)
}

func (q *taskQueue) insertBefore(task task, e *taskQueueElem) *taskQueueElem {
	if e.q != q {
		return nil
	}
	return q.insertTask(task, e.prev)
}

func (q *taskQueue) insertAfter(task task, e *taskQueueElem) *taskQueueElem {
	if e.q != q {
		return nil
	}
	return q.insertTask(task, e)
}

func (q *taskQueue) insert(e, at *taskQueueElem) *taskQueueElem {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.q = q
	q.len++
	return e
}

func (q *taskQueue) insertTask(task task, at *taskQueueElem) *taskQueueElem {
	return q.insert(&taskQueueElem{task: task}, at)
}

func (q *taskQueue) move(e, at *taskQueueElem) *taskQueueElem {
	if e == at {
		return e
	}

	e.prev.next = e.next
	e.next.prev = e.prev
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	return e
}

func (q *taskQueue) moveToFront(e *taskQueueElem) {
	if e.q != q || e == q.root.next {
		return
	}

	q.move(e, &q.root)
}

func (q *taskQueue) moveToBack(e *taskQueueElem) {
	if e.q != q || e == q.root.prev {
		return
	}

	q.move(e, q.root.prev)
}

func (q *taskQueue) moveBefore(e, mark *taskQueueElem) {
	if e.q != q || e == mark || mark.q != q {
		return
	}

	q.move(e, mark.prev)
}

func (q *taskQueue) moveAfter(e, mark *taskQueueElem) {
	if e.q != q || e == mark || mark.q != q {
		return
	}

	q.move(e, mark)
}

func (q *taskQueue) remove_(e *taskQueueElem) task {
	task := e.task
	e.prev.next = e.next
	e.next.prev = e.prev
	e.prev = nil
	e.next = nil
	e.q = nil
	e.task = nil
	q.len--
	return task
}

func (q *taskQueue) remove(e *taskQueueElem) task {
	if e == nil || e.q != q {
		return nil
	}

	return q.remove_(e)
}

func (q *taskQueue) popFrontTask() task {
	if q.len == 0 {
		return nil
	}

	task := q.root.next.task
	q.remove_(q.root.next)
	return task
}

func (q *taskQueue) popBackTask() task {
	if q.len == 0 {
		return nil
	}

	task := q.root.prev.task
	q.remove_(q.root.next)
	return task
}
