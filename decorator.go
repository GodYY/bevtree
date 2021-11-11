package bevtree

import "github.com/godyy/bevtree/internal/assert"

type decoratorTask logicTask

type decoratorTaskBase = oneChildTask

func newDecoratorTask(self decoratorTask) decoratorTaskBase {
	return newOneChildTask(self)
}

type decoratorNode = oneChildNode

type decoratorNodeBase = nodeOneChildBase

func newDecoratorNode(self decoratorNode) decoratorNodeBase {
	return newNodeOneChild(self)
}

// -----------------------------------------------------------
// Inverter
// -----------------------------------------------------------

type InverterNode struct {
	decoratorNodeBase
}

func NewInverter() *InverterNode {
	i := new(InverterNode)
	i.decoratorNodeBase = newDecoratorNode(i)
	return i
}

func (r *InverterNode) createTask(parent task) task {
	return ivtrTaskPool.get().(*inverterTask).ctr(r, parent)
}

func (r *InverterNode) destroyTask(t task) {
	t.(*inverterTask).dtr()
	ivtrTaskPool.put(t)
}

var ivtrTaskPool = newTaskPool(func() task { return newInverterTask() })

type inverterTask struct {
	decoratorTaskBase
}

func newInverterTask() *inverterTask {
	t := &inverterTask{}
	t.decoratorTaskBase = newDecoratorTask(t)
	return t
}

func (r *inverterTask) ctr(node *InverterNode, parent task) task {
	return r.decoratorTaskBase.ctr(node, parent)
}

func (r *inverterTask) onChildOver(child task, result Result, e *Env) Result {
	if result == RSuccess {
		return r.decoratorTaskBase.onChildOver(child, RFailure, e)
	} else {
		return r.decoratorTaskBase.onChildOver(child, RSuccess, e)
	}
}

// -----------------------------------------------------------
// Succeeder
// -----------------------------------------------------------

type SucceederNode struct {
	decoratorNodeBase
}

func NewSucceeder() *SucceederNode {
	s := new(SucceederNode)
	s.decoratorNodeBase = newDecoratorNode(s)
	return s
}

func (s *SucceederNode) createTask(parent task) task {
	return succTaskPool.get().(*succeederTask).ctr(s, parent)
}

func (s *SucceederNode) destroyTask(t task) {
	t.(*succeederTask).dtr()
	succTaskPool.put(t)
}

var succTaskPool = newTaskPool(func() task { return newSucceederTask() })

type succeederTask struct {
	decoratorTaskBase
}

func newSucceederTask() *succeederTask {
	t := &succeederTask{}
	t.decoratorTaskBase = newDecoratorTask(t)
	return t
}

func (t *succeederTask) ctr(node *SucceederNode, parent task) task {
	return t.decoratorTaskBase.ctr(node, parent)
}

func (t *succeederTask) onChildOver(child task, r Result, e *Env) Result {
	return t.decoratorTaskBase.onChildOver(child, RSuccess, e)
}

// -----------------------------------------------------------
// Repeater
// -----------------------------------------------------------

type RepeaterNode struct {
	decoratorNodeBase
	limited int
}

func NewRepeater(limited int) *RepeaterNode {
	assert.True(limited > 0, "invalid limited")

	r := new(RepeaterNode)
	r.decoratorNodeBase = newDecoratorNode(r)
	r.limited = limited
	return r
}

func (r *RepeaterNode) createTask(parent task) task {
	return reptrTaskPool.get().(*repeaterTask).ctr(r, parent)
}

func (r *RepeaterNode) destroyTask(t task) {
	t.(*repeaterTask).dtr()
	reptrTaskPool.put(t)
}

var reptrTaskPool = newTaskPool(func() task { return newRepeaterTask() })

type repeaterTask struct {
	decoratorTaskBase
	count int
}

func newRepeaterTask() *repeaterTask {
	t := &repeaterTask{}
	t.decoratorTaskBase = newDecoratorTask(t)
	return t
}

func (t *repeaterTask) ctr(node *RepeaterNode, parent task) task {
	return t.decoratorTaskBase.ctr(node, parent)
}

func (t *repeaterTask) getNode() *RepeaterNode {
	return t.node.(*RepeaterNode)
}

func (t *repeaterTask) onInit(e *Env) bool {
	node := t.getNode()
	if node.Child() == nil || node.limited <= 0 {
		return false
	}

	t.child = node.Child().createTask(t)
	e.pushCurrentTask(t.child)
	return true
}

func (t *repeaterTask) onUpdate(e *Env) Result {
	return RRunning
}

func (t *repeaterTask) onTerminate(e *Env) {
	t.count = 0
	t.decoratorTaskBase.onTerminate(e)
}

func (t *repeaterTask) onChildOver(child task, r Result, e *Env) Result {
	if debug {
		assert.Equal(child, t.child, "not child of it")
	}

	t.count++
	node := t.getNode()
	if t.count < node.limited && r == RSuccess {
		t.child = t.getNode().Child().createTask(t)
		e.pushCurrentTask(t.child)
		return RRunning
	} else {
		return r
	}
}

// -----------------------------------------------------------
// RepeatUntilFailNode
// -----------------------------------------------------------

type RepeatUntilFailNode struct {
	decoratorNodeBase
}

func NewRepeatUntilFail() *RepeatUntilFailNode {
	r := new(RepeatUntilFailNode)
	r.decoratorNodeBase = newDecoratorNode(r)
	return r
}

func (r *RepeatUntilFailNode) createTask(parent task) task {
	return rufTaskPool.get().(*repeatUntilFailTask).ctr(r, parent)
}

func (r *RepeatUntilFailNode) destroyTask(t task) {
	t.(*repeatUntilFailTask).dtr()
	rufTaskPool.put(t)
}

var rufTaskPool = newTaskPool(func() task { return newRepeatUntilFailTask() })

type repeatUntilFailTask struct {
	decoratorTaskBase
}

func newRepeatUntilFailTask() *repeatUntilFailTask {
	t := &repeatUntilFailTask{}
	t.decoratorTaskBase = newDecoratorTask(t)
	return t
}

func (t *repeatUntilFailTask) ctr(node *RepeatUntilFailNode, parent task) task {
	return t.decoratorTaskBase.ctr(node, parent)
}

func (t *repeatUntilFailTask) onChildOver(child task, r Result, e *Env) Result {
	if debug {
		assert.Equal(child, t.child, "not child of it")
	}

	if r == RSuccess {
		t.child = t.getNode().Child().createTask(t)
		e.pushCurrentTask(t.child)
		return RRunning
	}

	return RFailure
}
