package bevtree

type decoratorNode = logicNode

type decoratorNodeBase = nodeWithOneChild

func newDecoratorNode(self decoratorNode) decoratorNodeBase {
	return newNodeWithOneChild(self)
}

type InverterNode struct {
	decoratorNodeBase
}

func NewInverter() *InverterNode {
	i := new(InverterNode)
	i.decoratorNodeBase = newDecoratorNode(i)
	return i
}

func (r *InverterNode) onChildOver(child node, result Result, e *Env) Result {
	if result == RSuccess {
		return RFailure
	} else {
		return RSuccess
	}
}

type SucceederNode struct {
	decoratorNodeBase
}

func NewSucceeder() *SucceederNode {
	s := new(SucceederNode)
	s.decoratorNodeBase = newDecoratorNode(s)
	return s
}

func (s *SucceederNode) onChildOver(node, Result, *Env) Result {
	return RSuccess
}

type RepeaterNode struct {
	decoratorNodeBase
	limited int
	count   int
}

func NewRepeater(limited int) *RepeaterNode {
	if limited <= 0 {
		panic("invalid limited")
	}

	r := new(RepeaterNode)
	r.decoratorNodeBase = newDecoratorNode(r)
	r.limited = limited
	return r
}

func (r *RepeaterNode) onStart(e *Env) {
	if r.count < r.limited && r.Child() != nil {
		e.pushCurrentNode(r.Child())
	}
}

func (r *RepeaterNode) onUpdate(e *Env) Result {
	if r.count >= r.limited || r.Child() == nil {
		return RFailure
	} else {
		return RRunning
	}
}

func (r *RepeaterNode) onChildOver(child node, result Result, e *Env) Result {
	r.count++
	if r.count < r.limited && result == RSuccess {
		e.pushNextNode(r.Child())
		result = RRunning
	}

	return result
}

func (r *RepeaterNode) onEnd(e *Env) {
	r.count = 0
}

type RepeatUntilFailNode struct {
	decoratorNodeBase
}

func NewRepeatUntilFail() *RepeatUntilFailNode {
	r := new(RepeatUntilFailNode)
	r.decoratorNodeBase = newDecoratorNode(r)
	return r
}

func (r *RepeatUntilFailNode) onChildOver(child node, result Result, e *Env) Result {
	if result == RSuccess {
		e.pushNextNode(child)
		return RRunning
	}

	return RFailure
}
