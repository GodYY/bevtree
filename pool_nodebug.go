// +build !debug

package bevtree

func (p *taskPool) get() Task {
	return p.p.Get().(Task)
}

func (p *taskPool) put(task Task) {
	p.p.Put(task)
}

func getTaskQueElem() *taskQueElem {
	return taskElemPool.Get().(*taskQueElem)
}

func putTaskQueElem(e *taskQueElem) {
	taskElemPool.Put(e)
}
