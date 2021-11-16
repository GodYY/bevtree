package bevtree

import (
	"log"
	"reflect"

	"github.com/godyy/bevtree/internal/assert"
)

type nodeType int8

const (
	ntRoot = nodeType(iota)
	ntInverter
	ntSucceeder
	ntRepeater
	ntRepeatUntilFail
	ntSequence
	ntSelector
	ntRandSequence
	ntRandSelector
	ntParallel
	ntBehavior
)

var nodeTypeStrings = [...]string{
	ntRoot:            "root",
	ntInverter:        "inverter",
	ntSucceeder:       "succeeder",
	ntRepeater:        "repeater",
	ntRepeatUntilFail: "repeatuntilfail",
	ntSequence:        "sequence",
	ntSelector:        "selector",
	ntRandSequence:    "randsequence",
	ntRandSelector:    "randselector",
	ntParallel:        "parallel",
	ntBehavior:        "behavior",
}

var nodeString2Types = map[string]nodeType{
	"root":            ntRoot,
	"inverter":        ntInverter,
	"succeeder":       ntSucceeder,
	"repeater":        ntRepeater,
	"repeatuntilfail": ntRepeatUntilFail,
	"sequence":        ntSequence,
	"selector":        ntSelector,
	"randsequence":    ntRandSequence,
	"randselector":    ntRandSelector,
	"parallel":        ntParallel,
	"behavior":        ntBehavior,
}

func (t nodeType) String() string {
	return nodeTypeStrings[t]
}

type node interface {
	nodeType() nodeType
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

	// workflow
	createTask(parent task) task
	destroyTask(task)
}

type task interface {
	isBehavior() bool
	getParent() task
	detachParent()
	getStatus() status
	setQueElem(*taskQueElem)
	getQueElem() *taskQueElem
	isInQue() bool
	update(*Env) Result
	stop(*Env)
	lazyStop(*Env)
	childOver(task, Result, *Env) Result
	destroy()
}

type nodeBase struct {
	parent                   node
	prevSibling, nextSibling node
}

func newNode() nodeBase {
	return nodeBase{}
}

func (n *nodeBase) Parent() node { return n.parent }

func (n *nodeBase) setParent(parent node) {
	n.parent = parent
}

func (n *nodeBase) PrevSibling() node { return n.prevSibling }

func (n *nodeBase) setPrevSibling(node node) { n.prevSibling = node }

func (n *nodeBase) NextSibling() node { return n.nextSibling }

func (n *nodeBase) setNextSibling(node node) { n.nextSibling = node }

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

type taskBase struct {
	node             node
	parent           task
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

func (t *taskBase) ctr(node node, parent task) {
	if debug {
		assert.NotNilArg(node, "node")
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

func (t *taskBase) getParent() task {
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

func (t *taskBase) lazyStop(self task, e *Env) {
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
	task
	onInit(*Env) bool
	onUpdate(*Env) Result
	onTerminate(*Env)
	onStop(*Env)
	onLazyStop(*Env)
	onChildOver(task, Result, *Env) Result
}

type logicTaskBase struct {
	taskBase
	self logicTask
}

func newLogicTask(self logicTask) logicTaskBase {
	if debug {
		assert.NotNilArg(self != nil, "self")
	}

	return logicTaskBase{
		self: self,
	}
}

func (t *logicTaskBase) ctr(node node, parent task) task {
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
		assert.NotEqualf(st, sDestroyed, "%s.update: task already destroyed", reflect.TypeOf(t.self).Elem().Name())
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

func (t *logicTaskBase) childOver(child task, r Result, e *Env) Result {
	if t.getStatus() != sRunning {
		return RFailure
	}

	if debug {
		assert.Equal(child.getParent(), t.self, "invalid child")
		assert.NotEqualf(r, RRunning, "child:%s over with running", reflect.TypeOf(child).Elem().Name())
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
		assert.NotEqualf(t.getStatus(), sDestroyed, "%s already destroyed", reflect.TypeOf(t.self).Elem().Name())
		assert.NotEqualf(t.getStatus(), sRunning, "%s still running", reflect.TypeOf(t.self).Elem().Name())
	}

	t.setStatus(sDestroyed)
	t.node.destroyTask(t.self)
}

type oneChildNode interface {
	node
	Child() node
	SetChild(node)
}

type oneChildNodeBase struct {
	nodeBase
	self  node
	child node
}

func newNodeOneChild(self node) oneChildNodeBase {
	if debug {
		assert.NotNilArg(self, "self")
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

func (n *oneChildNodeBase) SetChild(child node) {
	assert.True(child != nil && child.Parent() == nil, "child nil or already has parent")

	if n.child != nil {
		n.child.setParent(nil)
		n.child = nil
	}

	if child != nil {
		child.setParent(n.self)
		n.child = child
	}
}

func (n *oneChildNodeBase) Child() node { return n.child }

func (n *oneChildNodeBase) FirstChild() node {
	return n.child
}

func (n *oneChildNodeBase) LastChild() node {
	return n.child
}

func (n *oneChildNodeBase) AddChild(child node) {
	assert.Nil(n.child, "already have child")
	assert.True(child != nil && child.Parent() == nil, "child nil or already has parent")

	n.SetChild(child)
}

func (n *oneChildNodeBase) AddChildBefore(child node, before node) {
}

func (n *oneChildNodeBase) AddChildAfter(child node, after node) {
}

func (n *oneChildNodeBase) RemoveChild(child node) {
	assert.Equal(child, n.child, "invalid child")

	n.SetChild(nil)
}

func (n *oneChildNodeBase) MoveChildBefore(child node, mark node) {
}

func (n *oneChildNodeBase) MoveChildAfter(child node, mark node) {
}

type oneChildTask struct {
	logicTaskBase
	child task
}

func newOneChildTask(self logicTask) oneChildTask {
	t := oneChildTask{}
	t.logicTaskBase = newLogicTask(self)
	return t
}

func (t *oneChildTask) ctr(node oneChildNode, parent task) task {
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

	t.child = node.createTask(t.self)
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

func (t *oneChildTask) onChildOver(child task, r Result, e *Env) Result {
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

func (rootNode) nodeType() nodeType { return ntRoot }

func (rootNode) Parent() node   { return nil }
func (rootNode) setParent(node) {}

func (rootNode) PrevSibling() node   { return nil }
func (rootNode) setPrevSibling(node) {}
func (rootNode) NextSibling() node   { return nil }
func (rootNode) setNextSibling(node) {}

func (r *rootNode) createTask(_ task) task {
	return rootTaskPool.get().(*rootTask).ctr(r)
}

func (r *rootNode) destroyTask(t task) {
	t.(*rootTask).dtr()
	rootTaskPool.put(t)
}

var rootTaskPool = newTaskPool(func() task { return newRootTask() })

type rootTask struct {
	oneChildTask
}

func newRootTask() *rootTask {
	t := &rootTask{}
	t.oneChildTask = newOneChildTask(t)
	return t
}

func (t *rootTask) ctr(node *rootNode) task {
	return t.oneChildTask.ctr(node, nil)
}

type BevTree struct {
	root_ *rootNode
}

func NewTree() *BevTree {
	tree := &BevTree{
		root_: newRoot(),
	}
	return tree
}

func (t *BevTree) root() *rootNode { return t.root_ }

func (t *BevTree) Clear() {
	t.root_.SetChild(nil)
}

func (t *BevTree) Update(e *Env) Result {
	if e.noTasks() {
		e.pushCurrentTask(t.root_.createTask(nil))
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

				assert.False(parent.isInQue(), "parent is over but in Que")

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

	assert.True(result == RRunning || e.noTasks(), "update over but already has tasks")

	return result
}

func (t *BevTree) Stop(e *Env) {
	e.reset()
}

var nodeCreators = [...]func() node{
	ntRoot:            func() node { return newRoot() },
	ntInverter:        func() node { return NewInverter() },
	ntSucceeder:       func() node { return NewSucceeder() },
	ntRepeater:        func() node { return newRepeater(0) },
	ntRepeatUntilFail: func() node { return NewRepeatUntilFail(false) },
	ntSequence:        func() node { return NewSequence() },
	ntSelector:        func() node { return NewSelector() },
	ntRandSequence:    func() node { return NewRandSequence() },
	ntRandSelector:    func() node { return NewRandSelector() },
	ntParallel:        func() node { return NewParallel() },
	ntBehavior:        func() node { return newBev() },
}

func createNode(nt nodeType) node {
	return nodeCreators[nt]()
}
