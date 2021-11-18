package bevtree

import (
	"log"
	"reflect"

	"github.com/GodYY/gutils/assert"
)

type NodeType int8

// Node metadata.
type NodeMETA struct {
	// node name.
	name string

	// type value.
	typ NodeType

	// function of creating node.
	creator func() Node
}

var nodeName2META = map[string]*NodeMETA{}
var nodeType2META = map[NodeType]*NodeMETA{}

func (t NodeType) String() string {
	meta := nodeType2META[t]
	assert.AssertF(meta != nil, "node type %d meta not found", t)
	return meta.name
}

// Register one node type.
func RegisterNodeType(name string, creator func() Node) NodeType {
	assert.NotEqual(name, "", "empty node type name")
	assert.AssertF(creator != nil, "node type \"%s\" creator nil", name)
	assert.AssertF(nodeName2META[name] == nil, "node type \"%s\" registered", name)

	meta := &NodeMETA{
		name:    name,
		typ:     NodeType(len(nodeName2META)),
		creator: creator,
	}

	nodeName2META[name] = meta
	nodeType2META[meta.typ] = meta

	return meta.typ
}

func createNode(nt NodeType) Node {
	meta := nodeType2META[nt]
	assert.AssertF(meta != nil, "node type %d meta not found", nt)
	return meta.creator()
}

var (
	root            = RegisterNodeType("root", func() Node { return newRoot() })
	inverter        = RegisterNodeType("inverter", func() Node { return NewInverter() })
	succeeder       = RegisterNodeType("succeeder", func() Node { return NewSucceeder() })
	repeater        = RegisterNodeType("repeater", func() Node { return newRepeater(0) })
	repeatUntilFail = RegisterNodeType("repeatuntilfail", func() Node { return NewRepeatUntilFail(true) })
	sequence        = RegisterNodeType("sequence", func() Node { return NewRandSequence() })
	selector        = RegisterNodeType("selector", func() Node { return NewSelector() })
	randSequence    = RegisterNodeType("randsequence", func() Node { return NewRandSequence() })
	randSelector    = RegisterNodeType("randselector", func() Node { return NewRandSelector() })
	parallel        = RegisterNodeType("parallel", func() Node { return NewParallel() })
	behavior        = RegisterNodeType("behavior", func() Node { return newBev() })
)

type Node interface {
	NodeType() NodeType
	Parent() Node
	setParent(Node)
	ChildCount() int
	FirstChild() Node
	LastChild() Node
	AddChild(Node)
	AddChildBefore(child, before Node)
	AddChildAfter(child, after Node)
	RemoveChild(Node)
	MoveChildBefore(child, mark Node)
	MoveChildAfter(child, mark Node)
	PrevSibling() Node
	setPrevSibling(Node)
	NextSibling() Node
	setNextSibling(Node)

	// workflow
	CreateTask(parent Task) Task
	DestroyTask(Task)
}

type status int8

const (
	sNone = status(iota)
	sRunning
	sTerminated
	sStopped
	sDestroyed
)

var statusStrings = [...]string{
	sNone:       "none",
	sRunning:    "running",
	sTerminated: "terminated",
	sStopped:    "stopped",
	sDestroyed:  "destroyed",
}

func (s status) String() string { return statusStrings[s] }

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

type Task interface {
	isBehavior() bool
	getParent() Task
	detachParent()
	getStatus() status
	setQueElem(*taskQueElem)
	getQueElem() *taskQueElem
	isInQue() bool
	update(*Env) Result
	stop(*Env)
	lazyStop(*Env)
	childOver(Task, Result, *Env) Result
	destroy()
}

type nodeBase struct {
	parent                   Node
	prevSibling, nextSibling Node
}

func newNode() nodeBase {
	return nodeBase{}
}

func (n *nodeBase) Parent() Node { return n.parent }

func (n *nodeBase) setParent(parent Node) {
	n.parent = parent
}

func (n *nodeBase) PrevSibling() Node { return n.prevSibling }

func (n *nodeBase) setPrevSibling(node Node) { n.prevSibling = node }

func (n *nodeBase) NextSibling() Node { return n.nextSibling }

func (n *nodeBase) setNextSibling(node Node) { n.nextSibling = node }

type taskBase struct {
	node             Node
	parent           Task
	latestUpdateSeri uint32
	st               status
	lzStop           lazyStop
	qElem            *taskQueElem
}

// func newTask(node node, parent task) taskBase {
// 	if debug {
// 		assertNilArg(node, "node")
// 	}

// 	return taskBase{node: node, parent: parent}
// }

func (t *taskBase) ctr(node Node, parent Task) {
	if debug {
		assert.Assert(node != nil, "node nil")
	}

	t.node = node
	t.parent = parent
	t.latestUpdateSeri = 0
	t.setStatus(sNone)
	t.setLZStop(lzsNone)
	t.qElem = nil
}

func (t *taskBase) dtr() {
	t.node = nil
	t.parent = nil
	t.qElem = nil
}

func (t *taskBase) getParent() Task {
	return t.parent
}

func (t *taskBase) detachParent() {
	t.parent = nil
}

func (t *taskBase) setStatus(st status) {
	t.st = st
}

func (t *taskBase) getStatus() status {
	return t.st
}

func (t *taskBase) setLZStop(lzStop lazyStop) {
	t.lzStop = lzStop
}

func (t *taskBase) getLZStop() lazyStop {
	return t.lzStop
}

func (t *taskBase) isLazyStop() bool {
	return t.lzStop != lzsNone
}

func (t *taskBase) setQueElem(e *taskQueElem) {
	t.qElem = e
}

func (t *taskBase) getQueElem() *taskQueElem {
	return t.qElem
}

func (t *taskBase) isInQue() bool { return t.qElem != nil }

func (t *taskBase) lazyStop(self Task, e *Env) {
	st := t.getStatus()
	log.Printf("%s.lazyStop %v", reflect.TypeOf(self).Elem().Name(), st)
	if st == sStopped || st == sTerminated || t.isLazyStop() {
		return
	}

	if t.latestUpdateSeri != e.getUpdateSeri() {
		t.setLZStop(lzsAfterUpdate)
	} else {
		t.setLZStop(lzsBeforeUpdate)
	}

	// assert.Nilf(t.qElem, "%s.qElem nil", reflect.TypeOf(self).Elem().Name())

	if t.qElem == nil || t.getLZStop() == lzsBeforeUpdate {
		e.pushCurrentTask(self)
	}
}

type logicTask interface {
	Task
	onInit(*Env) bool
	onUpdate(*Env) Result
	onTerminate(*Env)
	onStop(*Env)
	onLazyStop(*Env)
	onChildOver(Task, Result, *Env) Result
}

type logicTaskBase struct {
	taskBase
	self logicTask
}

func newLogicTask(self logicTask) logicTaskBase {
	if debug {
		assert.Assert(self != nil, "self nil")
	}

	return logicTaskBase{
		self: self,
	}
}

func (t *logicTaskBase) ctr(node Node, parent Task) Task {
	t.taskBase.ctr(node, parent)
	return t.self
}

func (t *logicTaskBase) isBehavior() bool { return false }

func (t *logicTaskBase) update(e *Env) Result {
	if debug {
		log.Printf("%s.update %v %v", reflect.TypeOf(t.self).Elem().Name(), t.getStatus(), t.getLZStop())
	}

	st := t.getStatus()

	if debug {
		assert.NotEqualF(st, sDestroyed, "%s.update: task already destroyed", reflect.TypeOf(t.self).Elem().Name())
	}

	// update seri.
	t.latestUpdateSeri = e.getUpdateSeri()

	// lazy stop before update.
	lzStop := t.getLZStop()
	if lzStop == lzsBeforeUpdate {
		return t.doLazyStop(e)
	}

	// init.
	if st == sNone && !t.self.onInit(e) {
		t.self.onTerminate(e)
		t.setStatus(sTerminated)
		return RFailure
	}

	// update.
	result := t.self.onUpdate(e)

	// lazy stop after update
	if lzStop == lzsAfterUpdate {
		return t.doLazyStop(e)
	}

	if result == RRunning {
		t.setStatus(sRunning)
	} else {
		// terminate.
		t.self.onTerminate(e)
		t.setStatus(sTerminated)
	}

	return result
}

func (t *logicTaskBase) lazyStop(e *Env) {
	t.taskBase.lazyStop(t.self, e)
}

func (t *logicTaskBase) doLazyStop(e *Env) Result {
	t.self.onLazyStop(e)
	t.self.onTerminate(e)
	t.setStatus(sStopped)
	t.setLZStop(lzsNone)
	return RFailure
}

func (t *logicTaskBase) stop(e *Env) {
	if t.getStatus() != sRunning {
		return
	}

	if debug {
		log.Printf("%s.stop", reflect.TypeOf(t.self).Elem().Name())
	}

	t.self.onStop(e)
	t.self.onTerminate(e)
	t.setStatus(sStopped)
	t.setLZStop(lzsNone)
}

func (t *logicTaskBase) childOver(child Task, r Result, e *Env) Result {
	if t.getStatus() != sRunning {
		return RFailure
	}

	if debug {
		assert.Equal(child.getParent(), t.self, "invalid child")
		assert.NotEqualF(r, RRunning, "child:%s over with running", reflect.TypeOf(child).Elem().Name())
	}

	if r = t.self.onChildOver(child, r, e); r != RRunning {
		t.self.onTerminate(e)
		t.setStatus(sNone)
		t.setLZStop(lzsNone)
	}

	return r
}

func (t *logicTaskBase) destroy() {
	if debug {
		assert.NotEqualF(t.getStatus(), sDestroyed, "%s already destroyed", reflect.TypeOf(t.self).Elem().Name())
		assert.NotEqualF(t.getStatus(), sRunning, "%s still running", reflect.TypeOf(t.self).Elem().Name())
	}

	t.setStatus(sDestroyed)
	t.node.DestroyTask(t.self)
}

type oneChildNode interface {
	Node
	Child() Node
	SetChild(Node)
}

type oneChildNodeBase struct {
	nodeBase
	self  Node
	child Node
}

func newNodeOneChild(self Node) oneChildNodeBase {
	if debug {
		assert.Assert(self != nil, "self")
	}

	return oneChildNodeBase{
		nodeBase: newNode(),
		self:     self,
	}
}

func (n *oneChildNodeBase) ChildCount() int {
	if n.child == nil {
		return 0
	}

	return 1
}

func (n *oneChildNodeBase) SetChild(child Node) {
	assert.Assert(child != nil && child.Parent() == nil, "child nil or already has parent")

	if n.child != nil {
		n.child.setParent(nil)
		n.child = nil
	}

	if child != nil {
		child.setParent(n.self)
		n.child = child
	}
}

func (n *oneChildNodeBase) Child() Node { return n.child }

func (n *oneChildNodeBase) FirstChild() Node {
	return n.child
}

func (n *oneChildNodeBase) LastChild() Node {
	return n.child
}

func (n *oneChildNodeBase) AddChild(child Node) {
	if n.child != nil {
		return
	}

	assert.Assert(child != nil && child.Parent() == nil, "child nil or already has parent")

	n.SetChild(child)
}

func (n *oneChildNodeBase) AddChildBefore(child Node, before Node) {
}

func (n *oneChildNodeBase) AddChildAfter(child Node, after Node) {
}

func (n *oneChildNodeBase) RemoveChild(child Node) {
	assert.Equal(child, n.child, "invalid child")

	n.SetChild(nil)
}

func (n *oneChildNodeBase) MoveChildBefore(child Node, mark Node) {
}

func (n *oneChildNodeBase) MoveChildAfter(child Node, mark Node) {
}

type oneChildTask struct {
	logicTaskBase
	child Task
}

func newOneChildTask(self logicTask) oneChildTask {
	t := oneChildTask{}
	t.logicTaskBase = newLogicTask(self)
	return t
}

func (t *oneChildTask) ctr(node oneChildNode, parent Task) Task {
	return t.logicTaskBase.ctr(node, parent)
}

func (t *oneChildTask) dtr() {
	if t.child != nil {
		if debug {
			log.Printf("%s.dtr() child not nil", reflect.TypeOf(t.self).Elem().Name())
		}

		t.child.destroy()
		t.child = nil
	}

	t.logicTaskBase.dtr()
}

func (t *oneChildTask) getNode() oneChildNode {
	return t.node.(oneChildNode)
}

func (t *oneChildTask) onInit(e *Env) bool {
	node := t.getNode().Child()
	if node == nil {
		return false
	}

	if debug {
		log.Printf("%s.onInit", reflect.TypeOf(t.self).Elem().Name())
	}

	t.child = node.CreateTask(t.self)
	e.pushCurrentTask(t.child)
	return true
}

func (t *oneChildTask) onUpdate(e *Env) Result {
	return RRunning
}

func (t *oneChildTask) onTerminate(e *Env) {
	t.child = nil
}

func (t *oneChildTask) onStop(e *Env) {
	t.child.detachParent()
}

func (t *oneChildTask) onLazyStop(e *Env) {
	if t.child != nil {
		t.child.lazyStop(e)
	}
}

func (t *oneChildTask) onChildOver(child Task, r Result, e *Env) Result {
	if debug {
		assert.Equal(child, t.child, "not child of it")
	}

	t.child = nil

	if t.isLazyStop() {
		return RRunning
	}

	return r
}

type rootNode struct {
	oneChildNodeBase
}

func newRoot() *rootNode {
	r := new(rootNode)
	r.oneChildNodeBase = newNodeOneChild(r)
	return r
}

func (rootNode) NodeType() NodeType { return root }

func (rootNode) Parent() Node   { return nil }
func (rootNode) setParent(Node) {}

func (rootNode) PrevSibling() Node   { return nil }
func (rootNode) setPrevSibling(Node) {}
func (rootNode) NextSibling() Node   { return nil }
func (rootNode) setNextSibling(Node) {}

func (r *rootNode) CreateTask(_ Task) Task {
	return rootTaskPool.get().(*rootTask).ctr(r)
}

func (r *rootNode) DestroyTask(t Task) {
	t.(*rootTask).dtr()
	rootTaskPool.put(t)
}

var rootTaskPool = newTaskPool(func() Task { return newRootTask() })

type rootTask struct {
	oneChildTask
}

func newRootTask() *rootTask {
	t := &rootTask{}
	t.oneChildTask = newOneChildTask(t)
	return t
}

func (t *rootTask) ctr(node *rootNode) Task {
	return t.oneChildTask.ctr(node, nil)
}

type BevTree struct {
	root_ *rootNode
}

func NewBevTree() *BevTree {
	tree := &BevTree{
		root_: newRoot(),
	}
	return tree
}

func (t *BevTree) Root() *rootNode { return t.root_ }

func (t *BevTree) Clear() {
	t.root_.SetChild(nil)
}

func (t *BevTree) Update(e *Env) Result {
	if e.noTasks() {
		e.pushCurrentTask(t.root_.CreateTask(nil))
	}

	e.update()

	result := RRunning
	for task := e.popCurrentTask(); task != nil; task = e.popCurrentTask() {
		r := task.update(e)
		st := task.getStatus()
		if st == sStopped {
			task.destroy()
			continue
		}

		if st == sTerminated {
			over := true
			for task.getParent() != nil {
				parent := task.getParent()
				if parent.getStatus() != sRunning {
					over = false
					break
				}

				r = parent.childOver(task, r, e)

				if r == RRunning {
					over = false
					break
				}

				assert.Assert(!parent.isInQue(), "parent is over but in Que")

				task.destroy()
				task = parent
			}

			task.destroy()

			if over {
				assert.Equal(result, RRunning, "update over reapeatedly")
				assert.NotEqual(r, RRunning, "update over with RRunning")

				result = r
			}
		} else if task.isBehavior() {
			e.pushNextTask(task)
		}
	}

	assert.Assert(result == RRunning || e.noTasks(), "update over but already has tasks")

	return result
}

func (t *BevTree) Stop(e *Env) {
	e.reset()
}
