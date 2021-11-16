package bevtree

import (
	"log"

	"github.com/godyy/bevtree/data"
	"github.com/godyy/bevtree/internal/assert"
	"github.com/godyy/bevtree/internal/finalize"
)

type Env struct {
	updateSeri         uint32
	taskQue            *taskQue
	taskUpdateBoundary *taskQueElem
	data.DataContext
	userData interface{}
}

func NewEnv(userData interface{}) *Env {
	e := &Env{
		taskQue:     newTaskQueue(),
		DataContext: data.NewBlackboard(),
		userData:    userData,
	}

	finalize.SetFinalizer(e)

	return e
}

func (e *Env) Release() {
	finalize.UnsetFinalizer(e)
	e.release()
}

func (e *Env) release() {
	e.clearTask()
	e.taskQue = nil
	e.taskUpdateBoundary = nil
	e.DataContext.Clear()
	e.DataContext = nil
	e.userData = nil
}

func (e *Env) Finalizer() {
	if debug {
		log.Println("Env.Finalizer")
	}
	e.release()
}

func (e *Env) reset() {
	e.updateSeri = 0
	e.clearTask()
	e.taskUpdateBoundary = nil
	e.DataContext.Clear()
}

func (e *Env) getTaskQue() *taskQue { return e.taskQue }

func (e *Env) DataCtx() data.DataContext { return e.DataContext }

func (e *Env) UserData() interface{} { return e.userData }

func (e *Env) getUpdateSeri() uint32 { return e.updateSeri }

func (e *Env) noTasks() bool {
	return e.taskQue.empty() || (e.taskQue.getLen() == 1 && e.taskQue.front() == e.taskUpdateBoundary)
}

func (e *Env) lazyPushUpdateBoundary() {
	if e.taskUpdateBoundary == nil {
		e.taskUpdateBoundary = e.taskQue.pushBack(nil)
	}
}

func (e *Env) pushTask(task task, nextRounds ...bool) {
	assert.NotNilArg(task, "task")

	e.lazyPushUpdateBoundary()

	nextRound := false
	if len(nextRounds) > 0 {
		nextRound = nextRounds[0]
	}

	elem := task.getQueElem()
	if elem != nil {
		if elem.q == e.taskQue {
			if nextRound {
				e.taskQue.moveToBack(elem)
			} else {
				e.taskQue.moveBefore(elem, e.taskUpdateBoundary)
			}
			return
		}
		elem.q.remove(elem)
	}

	if nextRound {
		elem = e.taskQue.pushBack(task)
	} else {
		elem = e.taskQue.insertBefore(task, e.taskUpdateBoundary)
	}
	task.setQueElem(elem)
}

func (e *Env) pushCurrentTask(task task) {
	e.pushTask(task)
}

func (e *Env) popCurrentTask() task {
	e.lazyPushUpdateBoundary()

	if e.taskQue.front() == e.taskUpdateBoundary {
		e.taskQue.moveToBack(e.taskUpdateBoundary)
		return nil
	}

	node := e.taskQue.popFrontTask()
	if node != nil {
		node.setQueElem(nil)
	}

	return node
}

func (e *Env) pushNextTask(task task) {
	e.pushTask(task, true)
}

func (e *Env) removeTask(task task) {
	elem := task.getQueElem()
	if elem != nil {
		e.taskQue.remove(elem)
		task.setQueElem(nil)
	}
}

func (e *Env) clearTask() {
	for !e.taskQue.empty() {
		task := e.taskQue.popFrontTask()
		if task == nil {
			continue
		}

		assert.True(task.isBehavior(), "task is not behavior")

		for task != nil {
			parent := task.getParent()
			task.stop(e)
			task.destroy()
			task = parent
		}
	}
}

func (e *Env) update() uint32 {
	e.lazyPushUpdateBoundary()
	e.updateSeri++
	return e.updateSeri
}
