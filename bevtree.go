package bevtree

import (
	"log"
	"reflect"

	"github.com/godyy/bevtree/internal/assert"
)

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
	getStatus() status
	getParent() task
	detachParent()
	setQueElem(*taskQueElem)
	getQueElem() *taskQueElem
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
		assert.NilArg(node, "node")
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

func (t *taskBase) isStopped() bool {
	return t.st == sStopped
}

func (t *taskBase) isOver() bool {
	return t.st == sNone
}

func (t *taskBase) isRunning() bool {
	return t.st == sRunning
}

func (t *taskBase) setQueElem(e *taskQueElem) {
	t.qElem = e
}

func (t *taskBase) getQueElem() *taskQueElem {
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
		assert.NilArg(self != nil, "self")
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

	if debug {
		log.Printf("%s.stop", reflect.TypeOf(t.self).Elem().Name())
	}

	t.self.onStop(e)
	t.self.onTerminate(e)
	t.setStatus(sStopped)
	t.setLZStop(lzsNone)
}

func (t *logicTaskBase) childOver(child task, r Result, e *Env) Result {
	if !t.isRunning() {
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
		assert.Falsef(t.isRunning(), "%s still running", reflect.TypeOf(t.self).Elem().Name())
	}

	t.setStatus(sDestroyed)
	t.node.destroyTask(t.self)
}

func (t *logicTaskBase) childLazyStop(child task, e *Env) {
	if child != nil && child.isRunning() {
		child.lazyStop(e)
		e.pushCurrentTask(child)
	}
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
		assert.NilArg(self, "self")
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

func (n *nodeOneChildBase) Child() node { return n.child }

func (n *nodeOneChildBase) FirstChild() node {
	return n.child
}

func (n *nodeOneChildBase) LastChild() node {
	return n.child
}

func (n *nodeOneChildBase) AddChild(child node) {
	assert.Nil(n.child, "already have child")
	assert.True(child != nil && child.Parent() == nil, "child nil or already has parent")

	n.SetChild(child)
}

func (n *nodeOneChildBase) AddChildBefore(child node, before node) {
}

func (n *nodeOneChildBase) AddChildAfter(child node, after node) {
}

func (n *nodeOneChildBase) RemoveChild(child node) {
	assert.Equal(child, n.child, "invalid child")

	n.SetChild(nil)
}

func (n *nodeOneChildBase) MoveChildBefore(child node, mark node) {
}

func (n *nodeOneChildBase) MoveChildAfter(child node, mark node) {
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
		t.childLazyStop(t.child, e)
		t.child = nil
	}
}

func (t *oneChildTask) onChildOver(child task, r Result, e *Env) Result {
	if debug {
		assert.Equal(child, t.child, "not child of it")
	}

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
		if task.isStopped() {
			task.destroy()
			continue
		}

		if task.isOver() {
			over := true
			for task.getParent() != nil {
				parent := task.getParent()
				if !parent.isRunning() {
					over = false
					break
				}

				r = parent.childOver(task, r, e)

				if parent.isRunning() {
					over = false
					break
				}

				if parent.getQueElem() != nil {
					e.removeTask(parent)
				}

				task.destroy()
				task = parent
			}

			task.destroy()

			if over {
				if debug {
					assert.Equal(result, RRunning, "update over reapeatedly")
					assert.NotEqual(r, RRunning, "update over with RRunning")
				}

				result = r
			}
		} else if task.isBehavior() {
			e.pushNextTask(task)
		}
	}

	if debug {
		assert.True(result == RRunning || e.noTasks(), "update over but already has tasks")
	}

	return result
}

func (t *BevTree) Reset(e *Env) {
	clearTaskQue(e.getTaskQue(), e)
	e.reset()
}

func clearTaskQue(q *taskQue, e *Env) {
	if debug {
		assert.NilArg(q, "taskQueue")
	}

	for !q.empty() {
		task := q.popFrontTask()
		if task == nil {
			continue
		}

		assert.True(task.isBehavior(), "task is not behavior")

		for task != nil {
			parent := task.getParent()
			task.stop(e)
			task.destroy()
			task = parent
		}
	}

}
