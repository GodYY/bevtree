package bevtree

type decoratorTask logicTask

type decoratorTaskBase = oneChildTask

func newDecoratorTask(self decoratorTask, node decoratorNode, parent task) decoratorTaskBase {
	return newOneChildTask(self, node, parent)
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
	return newInverterTask(r, parent)
}

func (r *InverterNode) destroyTask(t task) {}

type inverterTask struct {
	decoratorTaskBase
}

func newInverterTask(node decoratorNode, parent task) *inverterTask {
	t := &inverterTask{}
	t.decoratorTaskBase = newDecoratorTask(t, node, parent)
	return t
}

func (r *inverterTask) onChildOver(child task, result Result, e *Env) Result {
	if result == RSuccess {
		return RFailure
	} else {
		return RSuccess
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
	return newSucceederTask(s, parent)
}

func (s *SucceederNode) destroyTask(t task) {}

type succeederTask struct {
	decoratorTaskBase
}

func newSucceederTask(node decoratorNode, parent task) *succeederTask {
	t := &succeederTask{}
	t.decoratorTaskBase = newDecoratorTask(t, node, parent)
	return t
}

func (t *succeederTask) onChildOver(child task, r Result, e *Env) Result {
	return RSuccess
}

// -----------------------------------------------------------
// Repeater
// -----------------------------------------------------------

type RepeaterNode struct {
	decoratorNodeBase
	limited int
}

func NewRepeater(limited int) *RepeaterNode {
	assert(limited > 0, "invalid limited")

	r := new(RepeaterNode)
	r.decoratorNodeBase = newDecoratorNode(r)
	r.limited = limited
	return r
}

func (r *RepeaterNode) createTask(parent task) task {
	return newRepeaterTask(r, parent)
}

func (r *RepeaterNode) destroyTask(t task) {}

type repeaterTask struct {
	decoratorTaskBase
	count int
}

func newRepeaterTask(node decoratorNode, parent task) *repeaterTask {
	t := &repeaterTask{}
	t.decoratorTaskBase = newDecoratorTask(t, node, parent)
	return t
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
	t.count++
	node := t.getNode()
	if t.count < node.limited && r == RSuccess {
		e.pushCurrentTask(t.child)
		r = RRunning
	}

	return r
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
	return newRepeatUntilFailTask(r, parent)
}

func (r *RepeatUntilFailNode) destroyTask(t task) {}

type repeatUntilFailTask struct {
	decoratorTaskBase
}

func newRepeatUntilFailTask(node decoratorNode, parent task) *repeatUntilFailTask {
	t := &repeatUntilFailTask{}
	t.decoratorTaskBase = newDecoratorTask(t, node, parent)
	return t
}

func (t *repeatUntilFailTask) onChildOver(child task, r Result, e *Env) Result {
	if r == RSuccess {
		e.pushCurrentTask(child)
		return RRunning
	}

	return RFailure
}
