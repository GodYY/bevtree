package bevtree

import (
	"github.com/GodYY/gutils/assert"
)

type decoratorTask logicTask

type decoratorTaskBase = oneChildTask

func newDecoratorTask(self decoratorTask) decoratorTaskBase {
	return newOneChildTask(self)
}

type decoratorNode = oneChildNode

type decoratorNodeBase = oneChildNodeBase

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

func (r *InverterNode) NodeType() NodeType { return inverter }

func (r *InverterNode) CreateTask(parent Task) Task {
	return ivtrTaskPool.get().(*inverterTask).ctr(r, parent)
}

func (r *InverterNode) DestroyTask(t Task) {
	t.(*inverterTask).dtr()
	ivtrTaskPool.put(t)
}

var ivtrTaskPool = newTaskPool(func() Task { return newInverterTask() })

type inverterTask struct {
	decoratorTaskBase
}

func newInverterTask() *inverterTask {
	t := &inverterTask{}
	t.decoratorTaskBase = newDecoratorTask(t)
	return t
}

func (r *inverterTask) ctr(node *InverterNode, parent Task) Task {
	return r.decoratorTaskBase.ctr(node, parent)
}

func (r *inverterTask) onChildOver(child Task, result Result, e *Env) Result {
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

func (s *SucceederNode) NodeType() NodeType { return succeeder }

func (s *SucceederNode) CreateTask(parent Task) Task {
	return succTaskPool.get().(*succeederTask).ctr(s, parent)
}

func (s *SucceederNode) DestroyTask(t Task) {
	t.(*succeederTask).dtr()
	succTaskPool.put(t)
}

var succTaskPool = newTaskPool(func() Task { return newSucceederTask() })

type succeederTask struct {
	decoratorTaskBase
}

func newSucceederTask() *succeederTask {
	t := &succeederTask{}
	t.decoratorTaskBase = newDecoratorTask(t)
	return t
}

func (t *succeederTask) ctr(node *SucceederNode, parent Task) Task {
	return t.decoratorTaskBase.ctr(node, parent)
}

func (t *succeederTask) onChildOver(child Task, r Result, e *Env) Result {
	return t.decoratorTaskBase.onChildOver(child, RSuccess, e)
}

// -----------------------------------------------------------
// Repeater
// -----------------------------------------------------------

type RepeaterNode struct {
	decoratorNodeBase
	limited int
}

func newRepeater(limited int) *RepeaterNode {
	r := new(RepeaterNode)
	r.decoratorNodeBase = newDecoratorNode(r)
	r.limited = limited
	return r
}

func NewRepeater(limited int) *RepeaterNode {
	assert.Assert(limited > 0, "invalid limited")

	return newRepeater(limited)
}

func (r *RepeaterNode) NodeType() NodeType { return repeater }

func (r *RepeaterNode) CreateTask(parent Task) Task {
	return reptrTaskPool.get().(*repeaterTask).ctr(r, parent)
}

func (r *RepeaterNode) DestroyTask(t Task) {
	t.(*repeaterTask).dtr()
	reptrTaskPool.put(t)
}

var reptrTaskPool = newTaskPool(func() Task { return newRepeaterTask() })

type repeaterTask struct {
	decoratorTaskBase
	count int
}

func newRepeaterTask() *repeaterTask {
	t := &repeaterTask{}
	t.decoratorTaskBase = newDecoratorTask(t)
	return t
}

func (t *repeaterTask) ctr(node *RepeaterNode, parent Task) Task {
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

	t.child = node.Child().CreateTask(t)
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

func (t *repeaterTask) onChildOver(child Task, r Result, e *Env) Result {
	if debug {
		assert.Equal(child, t.child, "not child of it")
	}

	t.child = nil

	if t.isLazyStop() {
		return RRunning
	}

	t.count++
	node := t.getNode()
	if t.count < node.limited && r == RSuccess {
		t.child = t.getNode().Child().CreateTask(t)
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
	successOnFail bool
}

func NewRepeatUntilFail(successOnFail bool) *RepeatUntilFailNode {
	r := &RepeatUntilFailNode{successOnFail: successOnFail}
	r.decoratorNodeBase = newDecoratorNode(r)
	return r
}

func (r *RepeatUntilFailNode) NodeType() NodeType { return repeatUntilFail }

func (r *RepeatUntilFailNode) CreateTask(parent Task) Task {
	return rufTaskPool.get().(*repeatUntilFailTask).ctr(r, parent)
}

func (r *RepeatUntilFailNode) DestroyTask(t Task) {
	t.(*repeatUntilFailTask).dtr()
	rufTaskPool.put(t)
}

var rufTaskPool = newTaskPool(func() Task { return newRepeatUntilFailTask() })

type repeatUntilFailTask struct {
	decoratorTaskBase
}

func newRepeatUntilFailTask() *repeatUntilFailTask {
	t := &repeatUntilFailTask{}
	t.decoratorTaskBase = newDecoratorTask(t)
	return t
}

func (t *repeatUntilFailTask) ctr(node *RepeatUntilFailNode, parent Task) Task {
	return t.decoratorTaskBase.ctr(node, parent)
}

func (t *repeatUntilFailTask) getNode() *RepeatUntilFailNode {
	return t.decoratorTaskBase.getNode().(*RepeatUntilFailNode)
}

func (t *repeatUntilFailTask) onChildOver(child Task, r Result, e *Env) Result {
	if debug {
		assert.Equal(child, t.child, "not child of it")
	}

	t.child = nil

	if t.isLazyStop() {
		return RRunning
	}

	if r == RSuccess {
		t.child = t.getNode().Child().CreateTask(t)
		e.pushCurrentTask(t.child)
		return RRunning
	}

	if t.getNode().successOnFail {
		return RSuccess
	} else {
		return RFailure
	}
}
