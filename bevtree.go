package bevtree

import (
	"log"
	"reflect"
)

var debug = false

type status int8

const (
	sNone = status(iota)
	sRunning
	sStopped
	sDestroyed
)

var statusStrings = [...]string{
	sNone:      "none",
	sRunning:   "running",
	sStopped:   "stopped",
	sDestroyed: "destroyed",
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

	// workflow
	createTask(parent task) task
	destroyTask(task)
}

type task interface {
	isBehavior() bool
	isStopped() bool
	isOver() bool
	isRunning() bool
	getParent() task
	setQueElem(*taskQueueElem)
	getQueElem() *taskQueueElem
	update(*Env) Result
	stop(*Env)
	childOver(task, Result, *Env) Result
	lazyStop(*Env)
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

type taskBase struct {
	node             node
	parent           task
	latestUpdateSeri uint32
	st               status
	lzStop           lazyStop
	qElem            *taskQueueElem
}

func newTask(node node, parent task) taskBase {
	if debug {
		assertNilArg(node, "node")
	}

	return taskBase{node: node, parent: parent}
}

func (t *taskBase) getParent() task {
	return t.parent
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

func (t *taskBase) isStopped() bool {
	return t.st == sStopped
}

func (t *taskBase) isOver() bool {
	return t.st == sNone
}

func (t *taskBase) isRunning() bool {
	return t.st == sRunning
}

func (t *taskBase) setQueElem(e *taskQueueElem) {
	t.qElem = e
}

func (t *taskBase) getQueElem() *taskQueueElem {
	return t.qElem
}

func (t *taskBase) lazyStop(e *Env) {
	if !t.isRunning() {
		return
	}

	if t.latestUpdateSeri != e.getUpdateSeri() {
		t.setLZStop(lzsAfterUpdate)
	} else {
		t.setLZStop(lzsBeforeUpdate)
	}

	if debug {
		log.Printf("%s:lazyStop %v %d %d", reflect.TypeOf(t.node).Elem().Name(), t.lzStop, t.latestUpdateSeri, e.getUpdateSeri())
	}
}

type logicTask interface {
	task
	onInit(*Env) bool
	onUpdate(*Env) Result
	onTerminate(*Env)
	onLazyStop(*Env)
	onChildOver(task, Result, *Env) Result
}

type logicTaskBase struct {
	taskBase
	self logicTask
}

func newLogicTask(self logicTask, node node, parent task) logicTaskBase {
	if debug {
		assertNilArg(self != nil, "self")
		assertNilArg(node, "node")
	}

	return logicTaskBase{
		taskBase: newTask(node, parent),
		self:     self,
	}
}

func (t *logicTaskBase) isBehavior() bool { return false }

func (t *logicTaskBase) update(e *Env) Result {
	st := t.getStatus()

	if debug {
		assertF(st != sDestroyed, "%s.update: task already destroyed", reflect.TypeOf(t.self).Elem().Name())
	}

	// update seri.
	t.latestUpdateSeri = e.getUpdateSeri()

	// lazy stop before update.
	lzStop := t.getLZStop()
	if lzStop == lzsBeforeUpdate {
		return t.doLazyStop(e)
	}

	// init.
	if st != sRunning && !t.self.onInit(e) {
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
		t.setStatus(sNone)
	}

	return result
}

func (t *logicTaskBase) doLazyStop(e *Env) Result {
	if debug {
		log.Printf("%s.doLazyStop %v", reflect.TypeOf(t.self).Elem().Name(), t.getLZStop())
	}

	t.self.onLazyStop(e)
	t.self.onTerminate(e)
	t.setStatus(sStopped)
	t.setLZStop(lzsNone)
	return RFailure
}

func (t *logicTaskBase) stop(e *Env) {
	if !t.isRunning() {
		return
	}

	t.self.onTerminate(e)
	t.setStatus(sStopped)
	t.setLZStop(lzsNone)
}

func (t *logicTaskBase) childOver(child task, r Result, e *Env) Result {
	if !t.isRunning() {
		return RFailure
	}

	if debug {
		assert(child.getParent() == t.self, "invalid child")
		assertF(r != RRunning, "child:%s over with running", reflect.TypeOf(child).Elem().Name())
	}

	if r = t.self.onChildOver(child, r, e); r != RRunning {
		if debug {
			log.Printf("%s.childOver over with %v", reflect.TypeOf(t.self).Elem().Name(), r)
		}
		t.self.onTerminate(e)
		t.setStatus(sNone)
		t.setLZStop(lzsNone)
	}

	return r
}

func (t *logicTaskBase) destroy() {
	if t.getStatus() == sDestroyed {
		return
	}

	if debug {
		assertF(!t.isRunning(), "%s still running", reflect.TypeOf(t.self).Elem().Name())
	}

	t.node.destroyTask(t.self)
	t.node = nil
	t.setStatus(sDestroyed)
}

func (t *logicTaskBase) childLazyStop(child task, e *Env) {
	if child != nil && child.isRunning() {
		child.lazyStop(e)
		e.pushCurrentTask(child)
	}
}

type oneChildTask struct {
	logicTaskBase
	child task
}

func newOneChildTask(self logicTask, node oneChildNode, parent task) oneChildTask {
	t := oneChildTask{}
	t.logicTaskBase = newLogicTask(self, node, parent)
	return t
}

func (t *oneChildTask) getNode() oneChildNode {
	return t.node.(oneChildNode)
}

func (t *oneChildTask) onInit(e *Env) bool {
	if debug {
		log.Printf("%s.onInit", reflect.TypeOf(t.self).Elem().Name())
	}

	node := t.getNode().Child()
	if node == nil {
		return false
	}

	t.child = node.createTask(t.self)
	e.pushCurrentTask(t.child)
	return true
}

func (t *oneChildTask) onUpdate(e *Env) Result {
	if debug {
		log.Printf("%s.onUpdate", reflect.TypeOf(t.self).Elem().Name())
	}

	return RRunning
}

func (t *oneChildTask) onTerminate(e *Env) {}

func (t *oneChildTask) onLazyStop(e *Env) {
	t.childLazyStop(t.child, e)
}

type oneChildNode interface {
	node
	Child() node
	SetChild(node)
}

type nodeOneChildBase struct {
	nodeBase
	self  node
	child node
}

func newNodeOneChild(self node) nodeOneChildBase {
	if debug {
		assertNilArg(self, "self")
	}

	return nodeOneChildBase{
		nodeBase: newNode(),
		self:     self,
	}
}

func (n *nodeOneChildBase) ChildCount() int {
	if n.child == nil {
		return 0
	}

	return 1
}

func (n *nodeOneChildBase) SetChild(child node) {
	assert(child != nil && child.Parent() == nil, "child nil or already has parent")

	if n.child != nil {
		n.child.setParent(nil)
		n.child = nil
	}

	if child != nil {
		child.setParent(n.self)
		n.child = child
	}
}

func (n *nodeOneChildBase) Child() node { return n.child }

func (n *nodeOneChildBase) FirstChild() node {
	return n.child
}

func (n *nodeOneChildBase) LastChild() node {
	return n.child
}

func (n *nodeOneChildBase) AddChild(child node) {
	assert(n.child == nil, "already have child")
	assert(child != nil && child.Parent() == nil, "child nil or already has parent")

	n.SetChild(child)
}

func (n *nodeOneChildBase) AddChildBefore(child node, before node) {
}

func (n *nodeOneChildBase) AddChildAfter(child node, after node) {
}

func (n *nodeOneChildBase) RemoveChild(child node) {
	assert(child == n.child, "invalid child")

	n.SetChild(nil)
}

func (n *nodeOneChildBase) MoveChildBefore(child node, mark node) {
}

func (n *nodeOneChildBase) MoveChildAfter(child node, mark node) {
}

type rootTask struct {
	oneChildTask
}

func newRootTask(node oneChildNode) *rootTask {
	t := &rootTask{}
	t.oneChildTask = newOneChildTask(t, node, nil)
	return t
}

func (t *rootTask) onChildOver(child task, r Result, e *Env) Result {
	return r
}

type rootNode struct {
	nodeOneChildBase
}

func newRoot() *rootNode {
	r := new(rootNode)
	r.nodeOneChildBase = newNodeOneChild(r)
	return r
}

func (rootNode) Parent() node   { return nil }
func (rootNode) setParent(node) {}

func (rootNode) PrevSibling() node   { return nil }
func (rootNode) setPrevSibling(node) {}
func (rootNode) NextSibling() node   { return nil }
func (rootNode) setNextSibling(node) {}

func (r *rootNode) createTask(_ task) task {
	return newRootTask(r)
}

func (r *rootNode) destroyTask(t task) {}

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
		if task.isStopped() {
			continue
		}

		if task.isOver() {
			parent := task.getParent()
			for ; parent != nil && parent.isRunning(); task, parent = parent, parent.getParent() {
				r = parent.childOver(task, r, e)

				if parent.isRunning() {
					break
				} else if parent.getQueElem() != nil {
					e.removeTask(parent)
				}
			}

			if parent == nil {
				result = r
			}
		} else if task.isBehavior() {
			e.pushNextTask(task)
		}
	}

	if debug {
		assert(result == RRunning || e.noTasks(), "update over but already has tasks")
	}

	return result
}

func (t *BevTree) Reset(e *Env) {
	for task := e.popCurrentTask(); !e.noTasks(); task = e.popCurrentTask() {
		if task != nil {
			task.stop(e)
			for parent := task.getParent(); parent != nil && parent.isRunning(); parent = parent.getParent() {
				parent.stop(e)
			}
		}
	}

	e.reset()
}
