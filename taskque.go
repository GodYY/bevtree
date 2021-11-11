package bevtree

type taskQueElem struct {
	q    *taskQue
	task task
	prev *taskQueElem
	next *taskQueElem
}

func (e *taskQueElem) getNext() *taskQueElem {
	if p := e.next; e.q != nil && p != &e.q.root {
		return p
	}
	return nil
}

func (e *taskQueElem) getPrev() *taskQueElem {
	if p := e.prev; e.q != nil && p != &e.q.root {
		return p
	}
	return nil
}

type taskQue struct {
	root taskQueElem
	len  int
}

func newTaskQueue() *taskQue {
	return new(taskQue).init()
}

func (q *taskQue) init() *taskQue {
	if q.len > 0 {
		q.clear()
	} else {
		q.root.prev = &q.root
		q.root.next = &q.root
	}

	return q
}

func (q *taskQue) lazyInit() {
	if q.root.next == nil {
		q.init()
	}
}

func (q *taskQue) getLen() int { return q.len }

func (q *taskQue) empty() bool { return q.len == 0 }

func (q *taskQue) clear(fs ...func(task)) {
	var f func(task)
	if len(fs) > 0 {
		f = fs[0]
	}

	e := q.root.next
	for e != &q.root {
		if e.task != nil && f != nil {
			f(e.task)
		}

		next := e.next
		q.remove_(e)
		e = next
	}
}

func (q *taskQue) front() *taskQueElem {
	if q.len == 0 {
		return nil
	}

	return q.root.next
}

func (q *taskQue) back() *taskQueElem {
	if q.len == 0 {
		return nil
	}

	return q.root.prev
}

func (q *taskQue) frontTask() task {
	if q.len == 0 {
		return nil
	}
	return q.root.next.task
}

func (q *taskQue) backTask() task {
	if q.len == 0 {
		return nil
	}
	return q.root.prev.task
}

func (q *taskQue) pushFront(task task) *taskQueElem {
	q.lazyInit()
	return q.insertTask(task, &q.root)
}

func (q *taskQue) pushBack(task task) *taskQueElem {
	q.lazyInit()
	return q.insertTask(task, q.root.prev)
}

func (q *taskQue) insertBefore(task task, e *taskQueElem) *taskQueElem {
	if e.q != q {
		return nil
	}
	return q.insertTask(task, e.prev)
}

func (q *taskQue) insertAfter(task task, e *taskQueElem) *taskQueElem {
	if e.q != q {
		return nil
	}
	return q.insertTask(task, e)
}

func (q *taskQue) insert(e, at *taskQueElem) *taskQueElem {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.q = q
	q.len++
	return e
}

func (q *taskQue) insertTask(task task, at *taskQueElem) *taskQueElem {
	elem := getTaskQueElem()
	elem.task = task
	return q.insert(elem, at)
}

func (q *taskQue) move(e, at *taskQueElem) *taskQueElem {
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

func (q *taskQue) moveToFront(e *taskQueElem) {
	if e.q != q || e == q.root.next {
		return
	}

	q.move(e, &q.root)
}

func (q *taskQue) moveToBack(e *taskQueElem) {
	if e.q != q || e == q.root.prev {
		return
	}

	q.move(e, q.root.prev)
}

func (q *taskQue) moveBefore(e, mark *taskQueElem) {
	if e.q != q || e == mark || mark.q != q {
		return
	}

	q.move(e, mark.prev)
}

func (q *taskQue) moveAfter(e, mark *taskQueElem) {
	if e.q != q || e == mark || mark.q != q {
		return
	}

	q.move(e, mark)
}

func (q *taskQue) remove_(e *taskQueElem) task {
	task := e.task
	e.prev.next = e.next
	e.next.prev = e.prev
	e.prev = nil
	e.next = nil
	e.q = nil
	e.task = nil
	q.len--
	putTaskQueElem(e)
	return task
}

func (q *taskQue) remove(e *taskQueElem) task {
	if e == nil || e.q != q {
		return nil
	}

	return q.remove_(e)
}

func (q *taskQue) popFrontTask() task {
	if q.len == 0 {
		return nil
	}

	task := q.root.next.task
	q.remove_(q.root.next)
	return task
}

func (q *taskQue) popBackTask() task {
	if q.len == 0 {
		return nil
	}

	task := q.root.prev.task
	q.remove_(q.root.next)
	return task
}
