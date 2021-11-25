package bevtree

import (
	"github.com/GodYY/gutils/assert"
)

type DecoratorNode interface {
	Node
	Child() Node
	SetChild(Node)
}

type decoratorNode struct {
	node
	child Node
}

func newDecoratorNode() decoratorNode {
	return decoratorNode{
		node: newNode(),
	}
}

func (d *decoratorNode) Child() Node { return d.child }

func (d *decoratorNode) SetChild(self DecoratorNode, child Node) {
	assert.Assert(child == nil || child.Parent() == nil, "child already has parent")

	if d.child != nil {
		d.child.SetParent(nil)
		d.child = nil
	}

	if child != nil {
		child.SetParent(self)
		d.child = child
	}
}

type InverterNode struct {
	decoratorNode
}

func NewInverterNode() *InverterNode {
	return &InverterNode{decoratorNode: newDecoratorNode()}
}

func (i *InverterNode) NodeType() NodeType { return inverter }

func (i *InverterNode) SetChild(child Node) { i.decoratorNode.SetChild(i, child) }

type inverterTask struct {
	node *InverterNode
}

func (i *inverterTask) TaskType() TaskType { return TaskSerial }
func (i *inverterTask) OnCreate(node Node) { i.node = node.(*InverterNode) }
func (i *inverterTask) OnDestroy()         { i.node = nil }

func (i *inverterTask) OnInit(nextNodes *NodeList, ctx *Context) bool {
	if i.node.Child() == nil {
		return false
	} else {
		nextNodes.Push(i.node.Child())
		return true
	}
}

func (i *inverterTask) OnUpdate(ctx *Context) Result { return RRunning }
func (i *inverterTask) OnTerminate(ctx *Context)     {}

func (i *inverterTask) OnChildTerminated(result Result, _ *NodeList, ctx *Context) Result {
	if result == RSuccess {
		return RFailure
	} else {
		return RSuccess
	}
}

type SucceederNode struct {
	decoratorNode
}

func NewSucceederNode() *SucceederNode {
	return &SucceederNode{decoratorNode: newDecoratorNode()}
}

func (s *SucceederNode) NodeType() NodeType  { return succeeder }
func (s *SucceederNode) SetChild(child Node) { s.decoratorNode.SetChild(s, child) }

type succeederTask struct {
	node *SucceederNode
}

func (s *succeederTask) TaskType() TaskType { return TaskSerial }
func (s *succeederTask) OnCreate(node Node) { s.node = node.(*SucceederNode) }
func (s *succeederTask) OnDestroy()         { s.node = nil }

func (s *succeederTask) OnInit(nextNodes *NodeList, ctx *Context) bool {
	if s.node.Child() == nil {
		return false
	} else {
		nextNodes.Push(s.node.Child())
		return true
	}
}

func (s *succeederTask) OnUpdate(ctx *Context) Result { return RRunning }
func (s *succeederTask) OnTerminate(ctx *Context)     {}

func (s *succeederTask) OnChildTerminated(result Result, _ *NodeList, ctx *Context) Result {
	return RSuccess
}

type RepeaterNode struct {
	decoratorNode
	limited int
}

func newRepeaterNode() *RepeaterNode {
	return &RepeaterNode{
		decoratorNode: newDecoratorNode(),
	}
}

func NewRepeaterNode(limited int) *RepeaterNode {
	assert.Assert(limited > 0, "invalid limited")

	r := newRepeaterNode()
	r.limited = limited
	return r
}

func (r *RepeaterNode) NodeType() NodeType  { return repeater }
func (r *RepeaterNode) SetChild(child Node) { r.decoratorNode.SetChild(r, child) }

type repeaterTask struct {
	node  *RepeaterNode
	count int
}

func (r *repeaterTask) TaskType() TaskType { return TaskSerial }
func (r *repeaterTask) OnCreate(node Node) { r.node = node.(*RepeaterNode); r.count = 0 }
func (r *repeaterTask) OnDestroy()         { r.node = nil }

func (r *repeaterTask) OnInit(nextNodes *NodeList, ctx *Context) bool {
	if r.node.Child() == nil {
		return false
	} else {
		nextNodes.Push(r.node.Child())
		return true
	}
}

func (r *repeaterTask) OnUpdate(ctx *Context) Result { return RRunning }
func (r *repeaterTask) OnTerminate(ctx *Context)     {}

func (r *repeaterTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	r.count++
	if result != RFailure && r.count < r.node.limited {
		nextNodes.Push(r.node.Child())
		return RRunning
	} else {
		return result
	}
}

type RepeatUntilFailNode struct {
	decoratorNode
	successOnFail bool
}

func NewRepeatUntilFailNode(successOnFail bool) *RepeatUntilFailNode {
	return &RepeatUntilFailNode{
		decoratorNode: newDecoratorNode(),
		successOnFail: successOnFail,
	}
}

func (r *RepeatUntilFailNode) NodeType() NodeType  { return repeatUntilFail }
func (r *RepeatUntilFailNode) SetChild(child Node) { r.decoratorNode.SetChild(r, child) }

type repeatUntilFailTask struct {
	node *RepeatUntilFailNode
}

func (r *repeatUntilFailTask) TaskType() TaskType { return TaskSerial }
func (r *repeatUntilFailTask) OnCreate(node Node) { r.node = node.(*RepeatUntilFailNode) }
func (r *repeatUntilFailTask) OnDestroy()         { r.node = nil }

func (r *repeatUntilFailTask) OnInit(nextNodes *NodeList, ctx *Context) bool {
	if r.node.Child() == nil {
		return false
	} else {
		nextNodes.Push(r.node.Child())
		return true
	}
}
func (r *repeatUntilFailTask) OnUpdate(ctx *Context) Result { return RRunning }
func (r *repeatUntilFailTask) OnTerminate(ctx *Context)     {}

func (r *repeatUntilFailTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	if result == RSuccess {
		nextNodes.Push(r.node.Child())
		return RRunning
	} else if result == RFailure && r.node.successOnFail {
		return RSuccess
	} else {
		return result
	}
}
