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

func (d *decoratorNode) SetChild(child Node) bool {
	if child == nil || child.Parent() != nil {
		return false
	}

	if d.child != nil {
		d.child.SetParent(nil)
	}

	d.child = child

	return child != nil
}

type InverterNode struct {
	decoratorNode
}

func NewInverterNode() *InverterNode {
	return &InverterNode{decoratorNode: newDecoratorNode()}
}

func (i *InverterNode) NodeType() NodeType { return inverter }

func (i *InverterNode) SetChild(child Node) {
	if i.decoratorNode.SetChild(child) {
		child.SetParent(i)
	}
}

type inverterTask struct {
	node *InverterNode
}

func (i *inverterTask) TaskType() TaskType { return Serial }
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

func (i *inverterTask) OnUpdate(ctx *Context) Result { return Running }
func (i *inverterTask) OnTerminate(ctx *Context)     {}

func (i *inverterTask) OnChildTerminated(result Result, _ *NodeList, ctx *Context) Result {
	if result == Success {
		return Failure
	} else {
		return Success
	}
}

type SucceederNode struct {
	decoratorNode
}

func NewSucceederNode() *SucceederNode {
	return &SucceederNode{decoratorNode: newDecoratorNode()}
}

func (s *SucceederNode) NodeType() NodeType { return succeeder }

func (s *SucceederNode) SetChild(child Node) {
	if s.decoratorNode.SetChild(child) {
		child.SetParent(s)
	}
}

type succeederTask struct {
	node *SucceederNode
}

func (s *succeederTask) TaskType() TaskType { return Serial }
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

func (s *succeederTask) OnUpdate(ctx *Context) Result { return Running }
func (s *succeederTask) OnTerminate(ctx *Context)     {}

func (s *succeederTask) OnChildTerminated(result Result, _ *NodeList, ctx *Context) Result {
	return Success
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

func (r *RepeaterNode) NodeType() NodeType { return repeater }

func (r *RepeaterNode) SetChild(child Node) {
	if r.decoratorNode.SetChild(child) {
		child.SetParent(r)
	}
}

type repeaterTask struct {
	node  *RepeaterNode
	count int
}

func (r *repeaterTask) TaskType() TaskType { return Serial }
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

func (r *repeaterTask) OnUpdate(ctx *Context) Result { return Running }
func (r *repeaterTask) OnTerminate(ctx *Context)     {}

func (r *repeaterTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	r.count++
	if result != Failure && r.count < r.node.limited {
		nextNodes.Push(r.node.Child())
		return Running
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

func (r *RepeatUntilFailNode) NodeType() NodeType { return repeatUntilFail }

func (r *RepeatUntilFailNode) SetChild(child Node) {
	if r.decoratorNode.SetChild(child) {
		child.SetParent(r)
	}
}

type repeatUntilFailTask struct {
	node *RepeatUntilFailNode
}

func (r *repeatUntilFailTask) TaskType() TaskType { return Serial }
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
func (r *repeatUntilFailTask) OnUpdate(ctx *Context) Result { return Running }
func (r *repeatUntilFailTask) OnTerminate(ctx *Context)     {}

func (r *repeatUntilFailTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	if result == Success {
		nextNodes.Push(r.node.Child())
		return Running
	} else if result == Failure && r.node.successOnFail {
		return Success
	} else {
		return result
	}
}
