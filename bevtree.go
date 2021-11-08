package bevtree

import (
	"fmt"
)

type status int8

const (
	sNone = status(iota)
	sRunning
	sStopped
)

var statusStrings = [...]string{
	sNone:    "none",
	sRunning: "running",
	sStopped: "stopped",
}

type lazyStop int8

const (
	lzsNone = lazyStop(iota)
	lzsBeforeUpdate
	lzsAfterUpdate
)

var lazyStopStrings = [...]string{
	lzsNone:         "none",
	lzsBeforeUpdate: "before-update",
	lzsAfterUpdate:  "after-update",
}

func (l lazyStop) String() string { return lazyStopStrings[l] }

type Result int8

const (
	RSuccess = Result(iota)
	RFailure
	RRunning
)

var resultStrings = [...]string{
	RSuccess: "success",
	RFailure: "failure",
	RRunning: "running",
}

func (r Result) String() string { return resultStrings[r] }

type node interface {
	Parent() node
	setParent(node)
	ChildCount() int
	FirstChild() node
	LastChild() node
	AddChild(node)
	AddChildBefore(child, before node)
	AddChildAfter(child, after node)
	RemoveChild(node)
	MoveChildBefore(child, mark node)
	MoveChildAfter(child, mark node)
	PrevSibling() node
	setPrevSibling(node)
	NextSibling() node
	setNextSibling(node)
	isBehavior() bool

	// workflow
	setQueElem(*nodeQueueElem)
	getQueElem() *nodeQueueElem
	update(*Env) Result
	stop(*Env)
	lazyStop(*Env)
	childOver(node, Result, *Env) Result
	isStopped() bool
	isOver() bool
	isRunning() bool
}

type nodeBase struct {
	parent                   node
	prevSibling, nextSibling node
	status                   status
	latestUpdateSeri         uint32
	lzStop                   lazyStop
	qElem                    *nodeQueueElem
}

func (n *nodeBase) Parent() node { return n.parent }

func (n *nodeBase) setParent(parent node) {
	n.parent = parent
}

func (n *nodeBase) PrevSibling() node { return n.prevSibling }

func (n *nodeBase) setPrevSibling(node node) { n.prevSibling = node }

func (n *nodeBase) NextSibling() node { return n.nextSibling }

func (n *nodeBase) setNextSibling(node node) { n.nextSibling = node }

func (n *nodeBase) setQueElem(e *nodeQueueElem) {
	n.qElem = e
}

func (n *nodeBase) getQueElem() *nodeQueueElem {
	return n.qElem
}

func (n *nodeBase) lazyStop(e *Env) {
	if !n.isRunning() {
		return
	}

	if n.latestUpdateSeri != e.getUpdateSeri() {
		n.lzStop = lzsAfterUpdate
	} else {
		n.lzStop = lzsBeforeUpdate
	}
}

func (n *nodeBase) isStopped() bool { return n.status == sStopped }

func (n *nodeBase) isOver() bool { return n.status == sNone }

func (n *nodeBase) isRunning() bool { return n.status == sRunning }

type logicNode interface {
	node
	onStart(*Env)
	onUpdate(*Env) Result
	onEnd(*Env)
	onStop(*Env)
	onLazyStop(*Env)
	onChildOver(node, Result, *Env) Result
}

type logicNodeBase struct {
	nodeBase
	self logicNode
}

func newLogicNode(self logicNode) logicNodeBase {
	if self == nil {
		panic("nil self")
	}

	return logicNodeBase{
		self: self,
	}
}

func (n *logicNodeBase) isBehavior() bool {
	return false
}

func (n *logicNodeBase) update(e *Env) Result {
	n.latestUpdateSeri = e.getUpdateSeri()

	if n.status != sRunning {
		n.self.onStart(e)
	}

	if n.lzStop == lzsBeforeUpdate {
		return n.doLazyStop(e)
	}

	result := n.self.onUpdate(e)

	if n.lzStop == lzsAfterUpdate {
		return n.doLazyStop(e)
	}

	if result == RRunning {
		n.status = sRunning
	} else {
		n.self.onEnd(e)
		n.status = sNone
	}

	return result
}

func (n *logicNodeBase) stop(e *Env) {
	if n.status == sNone || n.status == sStopped {
		return
	}

	n.self.onStop(e)
	n.self.onEnd(e)
	n.status = sStopped
	n.lzStop = lzsNone
}

func (n *logicNodeBase) onStop(*Env) {}

func (n *logicNodeBase) doLazyStop(e *Env) Result {
	fmt.Println("logic doLazyStop", n.lzStop)
	n.self.onLazyStop(e)
	n.self.onEnd(e)
	n.status = sStopped
	n.lzStop = lzsNone
	return RFailure
}

func (n *logicNodeBase) childOver(child node, result Result, e *Env) Result {
	if !n.isRunning() {
		return RFailure
	}

	if child.Parent() != n.self {
		panic("child not belongs to")
	}

	if result == RRunning {
		panic("child completed with Running")
	}

	if result = n.self.onChildOver(child, result, e); result != RRunning {
		n.self.onEnd(e)
		n.status = sNone
	}

	return result
}

func (n *logicNodeBase) childLazyStop(child node, e *Env) {
	if child.isRunning() {
		child.lazyStop(e)
		e.pushCurrentNode(child)
	}
}

type nodeWithOneChild struct {
	logicNodeBase
	child node
}

func newNodeWithOneChild(self logicNode) nodeWithOneChild {
	return nodeWithOneChild{
		logicNodeBase: newLogicNode(self),
	}
}

func (n *nodeWithOneChild) ChildCount() int {
	if n.child == nil {
		return 0
	}

	return 1
}

func (n *nodeWithOneChild) SetChild(child node) {
	if child == n.child {
		return
	}

	if child != nil && child.Parent() != nil {
		panic("child has parent")
	}

	if n.child != nil {
		n.child.setParent(nil)
		n.child = nil
	}

	if child != nil {
		child.setParent(n.self)
		n.child = child
	}
}

func (n *nodeWithOneChild) Child() node { return n.child }

func (n *nodeWithOneChild) FirstChild() node {
	return n.child
}

func (n *nodeWithOneChild) LastChild() node {
	return n.child
}

func (n *nodeWithOneChild) AddChild(child node) {
	if child == nil || child.Parent() != nil {
		panic("invalid child")
	}

	n.SetChild(child)
}

func (n *nodeWithOneChild) AddChildBefore(child node, before node) {
}

func (n *nodeWithOneChild) AddChildAfter(child node, after node) {
}

func (n *nodeWithOneChild) RemoveChild(child node) {
	if child != n.child {
		panic("child missmatch")
	}

	n.SetChild(nil)
}

func (n *nodeWithOneChild) MoveChildBefore(child node, mark node) {
}

func (n *nodeWithOneChild) MoveChildAfter(child node, mark node) {
}

func (n *nodeWithOneChild) onStart(e *Env) {
	if n.child != nil {
		e.pushCurrentNode(n.child)
	}
}

func (n *nodeWithOneChild) onUpdate(*Env) Result {
	if n.child == nil {
		return RFailure
	}

	return RRunning
}

func (n *nodeWithOneChild) onLazyStop(e *Env) {
	if n.child.isRunning() {
		n.childLazyStop(n.child, e)
	}
}

func (n *nodeWithOneChild) onChildOver(child node, result Result, e *Env) Result {
	// if child != n.child {
	// 	panic("child missmatch")
	// }

	return result
}

func (n *nodeWithOneChild) onEnd(*Env) {}

type rootNode struct {
	nodeWithOneChild
}

func newRoot() *rootNode {
	r := new(rootNode)
	r.nodeWithOneChild = newNodeWithOneChild(r)
	return r
}

func (rootNode) Parent() node   { return nil }
func (rootNode) setParent(node) {}

func (rootNode) PrevSibling() node   { return nil }
func (rootNode) setPrevSibling(node) {}
func (rootNode) NextSibling() node   { return nil }
func (rootNode) setNextSibling(node) {}

type BevTree struct {
	_root *rootNode
}

func NewTree() *BevTree {
	tree := &BevTree{
		_root: newRoot(),
	}
	return tree
}

func (t *BevTree) root() *rootNode { return t._root }

func (t *BevTree) Clear() {
	t._root.SetChild(nil)
}

func (t *BevTree) Update(e *Env) Result {
	if e.noNodes() {
		e.pushCurrentNode(t._root)
	}

	e.update()

	result := RRunning
	for node := e.popCurrentNode(); node != nil; node = e.popCurrentNode() {
		result = node.update(e)
		if node.isStopped() {
			continue
		}

		if node.isOver() {
			for parent := node.Parent(); parent != nil && result != RRunning; node, parent = parent, parent.Parent() {
				result = parent.childOver(node, result, e)
			}
		} else if node.isBehavior() {
			e.pushNextNode(node)
		}
	}

	return result
}

func (t *BevTree) Reset(e *Env) {
	for node := e.popCurrentNode(); !e.noNodes(); node = e.popCurrentNode() {
		if node != nil {
			node.stop(e)
			for parent := node.Parent(); parent != nil && parent.isRunning(); parent = parent.Parent() {
				parent.stop(e)
			}
		}
	}

	e.reset()
}
