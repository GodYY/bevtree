// +build !debug

package bevtree

func (p *taskPool) get() task {
	return p.p.Get().(task)
}

func (p *taskPool) put(task task) {
	p.p.Put(task)
}

func getTaskQueElem() *taskQueElem {
	return taskElemPool.Get().(*taskQueElem)
}

func putTaskQueElem(e *taskQueElem) {
	taskElemPool.Put(e)
}
