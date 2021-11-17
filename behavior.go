package bevtree

import (
	"log"

	"github.com/GodYY/gutils/assert"
)

type BevInst interface {
	OnInit(*Env)
	OnUpdate(*Env) Result
	OnTerminate(*Env)
}

// Bev defines a behavior, it can create and destroy instances
// of it.
type Bev interface {
	CreateBev() BevInst
	DestroyBev(BevInst)
}

type BevNode struct {
	nodeBase
	bev Bev
}

func newBev() *BevNode {
	return &BevNode{}
}

func NewBev(bev Bev) *BevNode {
	assert.NotNilArg(bev, "bev")

	return &BevNode{
		bev: bev,
	}
}

func (BevNode) nodeType() nodeType        { return ntBehavior }
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
	bev := b.bev.CreateBev()
	assert.NotNil(bev, "bev create nil bevInst")

	return bevTaskPool.get().(*bevTask).ctr(b, parent, bev)
}

func (b *BevNode) destroyTask(t task) {
	bt := t.(*bevTask)
	b.bev.DestroyBev(bt.bev)
	bt.dtr()
	bevTaskPool.put(t)
}

var bevTaskPool = newTaskPool(func() task { return newBevTask() })

type bevTask struct {
	taskBase
	bev BevInst
}

func newBevTask() *bevTask {
	return new(bevTask)
}

func (t *bevTask) ctr(node *BevNode, parent task, bev BevInst) task {
	assert.NotNilArg(bev, "bev")

	t.taskBase.ctr(node, parent)
	t.bev = bev

	return t
}

func (t *bevTask) dtr() {
	t.bev = nil
	t.taskBase.dtr()
}

func (t *bevTask) isBehavior() bool { return true }

func (t *bevTask) update(e *Env) Result {
	if debug {
		log.Printf("bevTask.update %v %v", t.getStatus(), t.getLZStop())
	}

	st := t.getStatus()

	if debug {
		assert.NotEqual(st, sDestroyed, "bevTask already destroyed")
	}

	// update seri.
	t.latestUpdateSeri = e.getUpdateSeri()

	lzStop := t.getLZStop()

	// lazy stop before update.
	if lzStop == lzsBeforeUpdate {
		return t.doLazyStop(e)
	}

	// init.
	if st == sNone {
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
		t.setStatus(sTerminated)
	}

	return result
}

func (t *bevTask) stop(e *Env) {
	if t.getStatus() != sRunning {
		return
	}

	t.bev.OnTerminate(e)
	t.setStatus(sStopped)
	t.setLZStop(lzsNone)
}

func (t *bevTask) lazyStop(e *Env) {
	t.taskBase.lazyStop(t, e)
}

func (t *bevTask) doLazyStop(e *Env) Result {
	t.bev.OnTerminate(e)
	t.setStatus(sStopped)
	t.setLZStop(lzsNone)
	return RFailure
}

func (t *bevTask) childOver(_ task, _ Result, _ *Env) Result {
	panic("should not be called")
}

func (t *bevTask) destroy() {
	if debug {
		assert.NotEqual(t.getStatus(), sDestroyed, "bevTask already destroyed")
		assert.NotEqual(t.getStatus(), sRunning, "bevTask still running")
	}

	t.st = sDestroyed
	t.node.destroyTask(t)
}