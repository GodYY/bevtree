package bevtree

import (
	"log"
)

type Bev interface {
	OnInit(*Env)
	OnUpdate(*Env) Result
	OnTerminate(*Env)
}

type BevDefiner interface {
	CreateBev() Bev
}

type BevNode struct {
	nodeBase
	bev BevDefiner
}

func NewBev(bevDef BevDefiner) *BevNode {
	assertNilArg(bevDef, "bevDef")

	return &BevNode{
		bev: bevDef,
	}
}

func (BevNode) ChildCount() int           { return 0 }
func (BevNode) AddChild(node)             {}
func (BevNode) RemoveChild(node)          {}
func (BevNode) AddChildBefore(_, _ node)  {}
func (BevNode) AddChildAfter(_, _ node)   {}
func (BevNode) MoveChildBefore(_, _ node) {}
func (BevNode) MoveChildAfter(_, _ node)  {}
func (BevNode) FirstChild() node          { return nil }
func (BevNode) LastChild() node           { return nil }

func (b *BevNode) createTask(parent task) task {
	return newBevTask(b, parent, b.bev.CreateBev())
}

func (b *BevNode) destroyTask(t task) {}

type bevTask struct {
	taskBase
	bev Bev
}

func newBevTask(node *BevNode, parent task, bev Bev) *bevTask {
	assertNilArg(bev, "bev")

	t := &bevTask{
		bev:      bev,
		taskBase: newTask(node, parent),
	}

	return t
}

func (t *bevTask) isBehavior() bool { return true }

func (t *bevTask) update(e *Env) Result {
	st := t.getStatus()

	if debug {
		assert(st != sDestroyed, "bevTask already destroyed")
	}

	// update seri.
	t.latestUpdateSeri = e.getUpdateSeri()

	lzStop := t.getLZStop()

	// lazy stop before update.
	if lzStop == lzsBeforeUpdate {
		return t.doLazyStop(e)
	}

	// init.
	if st != sRunning {
		t.bev.OnInit(e)
	}

	// update.
	result := t.bev.OnUpdate(e)

	// lazy stop after update.
	if lzStop == lzsAfterUpdate {
		return t.doLazyStop(e)
	}

	if result == RRunning {
		t.setStatus(sRunning)
	} else {
		// terminate.
		t.bev.OnTerminate(e)
		t.setStatus(sNone)
	}

	return result
}

func (t *bevTask) stop(e *Env) {
	if !t.isRunning() {
		return
	}

	t.bev.OnTerminate(e)
	t.setStatus(sStopped)
	t.setLZStop(lzsNone)
}

func (t *bevTask) doLazyStop(e *Env) Result {
	if debug {
		log.Println("bevTask.doLazyStop", t.getLZStop())
	}

	t.stop(e)
	return RFailure
}

func (t *bevTask) childOver(_ task, _ Result, _ *Env) Result {
	panic("should not be called")
}

func (t *bevTask) destroy() {
	st := t.getStatus()
	if st == sDestroyed {
		return
	}

	if debug {
		assert(st != sRunning, "bevTask still running")
	}

	t.node.destroyTask(t)
	t.node = nil
	t.st = sDestroyed
}
