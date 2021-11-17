package bevtree

import (
	"log"
	"math/rand"
	"reflect"

	"github.com/GodYY/gutils/assert"
)

type compositeNode = node

type compositeTask = logicTask

type compositeNodeBase struct {
	nodeBase
	self       compositeNode
	childCount int
	firstChild node
	lastChild  node
}

func newCompositeNode(self compositeNode) compositeNodeBase {
	if debug {
		assert.NotNilArg(self, "self")
	}

	return compositeNodeBase{
		nodeBase: newNode(),
		self:     self,
	}
}

func (c *compositeNodeBase) ChildCount() int { return c.childCount }

func (c *compositeNodeBase) FirstChild() node { return c.firstChild }

func (c *compositeNodeBase) LastChild() node { return c.lastChild }

func (c *compositeNodeBase) AddChild(child node) {
	assert.True(child != nil && child.Parent() == nil, "child nil or already has parent")

	if c.lastChild == nil {
		c.lastChild = child
		c.firstChild = child
		child.setParent(c.self)
		c.childCount++
	} else {
		c.AddChildAfter(child, c.lastChild)
	}
}

func (c *compositeNodeBase) AddChildBefore(child, mark node) {
	assert.True(child != nil && child.Parent() == nil, "child nil or already has parent")
	assert.True(mark != nil && mark.Parent() == c.self, "mark nil or not child of it")

	child.setParent(c.self)

	if prev := mark.PrevSibling(); prev != nil {
		prev.setNextSibling(child)
		child.setPrevSibling(prev)
	} else {
		child.setPrevSibling(nil)
		c.firstChild = child
	}

	child.setNextSibling(mark)
	mark.setPrevSibling(child)

	c.childCount++
}

func (c *compositeNodeBase) AddChildAfter(child, mark node) {
	assert.True(child != nil && child.Parent() == nil, "child nil or already has parent")
	assert.True(mark != nil && mark.Parent() == c.self, "mark nil or not child of it")

	child.setParent(c.self)

	if next := mark.NextSibling(); next != nil {
		next.setPrevSibling(child)
		child.setNextSibling(next)
	} else {
		child.setNextSibling(nil)
		c.lastChild = child
	}

	mark.setNextSibling(child)
	child.setPrevSibling(mark)

	c.childCount++
}

func (c *compositeNodeBase) MoveChildBefore(child, mark node) {
	assert.True(child != nil && child.Parent() == c.self, "child nil or not child of it")
	assert.True(mark != nil && mark.Parent() == c.self, "mark nil or not child of it")

	if child.PrevSibling() != nil {
		child.PrevSibling().setNextSibling(child.NextSibling())
	}

	if child.NextSibling() != nil {
		child.NextSibling().setPrevSibling(child.PrevSibling())
	}

	if mark.PrevSibling() != nil {
		mark.PrevSibling().setNextSibling(child)
		child.setPrevSibling(mark.PrevSibling())
	} else {
		child.setPrevSibling(nil)
		c.firstChild = child
	}

	mark.setPrevSibling(child)
	child.setNextSibling(mark)
}

func (c *compositeNodeBase) MoveChildAfter(child, mark node) {
	assert.True(child != nil && child.Parent() == c.self, "child nil or not child of it")
	assert.True(mark != nil && mark.Parent() == c.self, "mark nil or not child of it")

	if child.PrevSibling() != nil {
		child.PrevSibling().setNextSibling(child.NextSibling())
	}

	if child.NextSibling() != nil {
		child.NextSibling().setPrevSibling(child.PrevSibling())
	}

	if mark.NextSibling() != nil {
		mark.NextSibling().setPrevSibling(child)
		child.setNextSibling(mark.NextSibling())
	} else {
		child.setNextSibling(nil)
		c.lastChild = child
	}

	mark.setNextSibling(child)
	child.setPrevSibling(mark)
}

func (c *compositeNodeBase) RemoveChild(child node) {
	assert.True(child != nil && child.Parent() == c.self, "child nil or not child of it")

	if child.PrevSibling() != nil {
		child.PrevSibling().setNextSibling(child.NextSibling())
	} else {
		c.firstChild = child.NextSibling()
	}

	if child.NextSibling() != nil {
		child.NextSibling().setPrevSibling(child.PrevSibling())
	} else {
		c.lastChild = child.PrevSibling()
	}

	child.setNextSibling(nil)
	child.setPrevSibling(nil)
	child.setParent(nil)
	c.childCount--
}

type compositeTaskBase struct {
	logicTaskBase
}

func newCompisteTask(self compositeTask) compositeTaskBase {
	return compositeTaskBase{
		logicTaskBase: newLogicTask(self),
	}
}

func (t *compositeTaskBase) getNode() compositeNode {
	return t.node
}

// -----------------------------------------------------------
// CustomSequence
// -----------------------------------------------------------

type customSeqTask struct {
	compositeTaskBase
	curChild    task
	curNode     node
	keepOn      func(Result) bool
	getNextNode func() node
}

func newCustomSeqTask(self compositeTask, keepOn func(Result) bool, getNextNode func() node) customSeqTask {
	if debug {
		assert.NotNilArg(keepOn, "keepOn")
		assert.NotNilArg(getNextNode, "getNextChild")
	}

	return customSeqTask{
		compositeTaskBase: newCompisteTask(self),
		keepOn:            keepOn,
		getNextNode:       getNextNode,
	}
}

func (t *customSeqTask) dtr() {
	if t.curChild != nil {
		if debug {
			log.Printf("%s.dtr() curChild not nil", reflect.TypeOf(t.self).Elem().Name())
		}

		t.curChild.destroy()
		t.curChild = nil
	}

	t.curNode = nil
	t.compositeTaskBase.dtr()
}

func (t *customSeqTask) onInit(e *Env) bool {
	t.curNode = t.getNextNode()
	if t.curNode == nil {
		return false
	}

	t.curChild = t.curNode.createTask(t.self)
	e.pushCurrentTask(t.curChild)
	return true
}

func (t *customSeqTask) onUpdate(e *Env) Result {
	return RRunning
}

func (t *customSeqTask) onTerminate(e *Env) {
	t.curChild = nil
	t.curNode = nil
}

func (t *customSeqTask) onStop(e *Env) {
	t.curChild.detachParent()
}

func (t *customSeqTask) onLazyStop(e *Env) {
	if t.curChild != nil {
		t.curChild.lazyStop(e)
	}
}

func (t *customSeqTask) onChildOver(child task, r Result, e *Env) Result {
	assert.Equal(child, t.curChild, "not current child")

	t.curChild = nil

	if t.isLazyStop() {
		return RRunning
	}

	if !t.keepOn(r) {
		return r
	}

	nextNode := t.getNextNode()
	if nextNode == nil {
		return r
	}

	t.curNode = nextNode
	t.curChild = nextNode.createTask(t.self)
	e.pushCurrentTask(t.curChild)
	return RRunning
}

// -----------------------------------------------------------
// ChildSequence
// -----------------------------------------------------------

type childSeqTask struct {
	customSeqTask
}

func newChildSeqTask(self compositeTask, keepOn func(Result) bool) *childSeqTask {
	t := new(childSeqTask)
	t.customSeqTask = newCustomSeqTask(self, keepOn, t.getNextNode)
	return t
}

func (t *childSeqTask) getNextNode() node {
	if t.curNode == nil {
		return t.getNode().FirstChild()
	} else {
		return t.curNode.NextSibling()
	}
}

// -----------------------------------------------------------
// Sequence
// -----------------------------------------------------------

func sequenceKeepOn(r Result) bool {
	return r == RSuccess
}

type SequenceNode struct {
	compositeNodeBase
}

func NewSequence() *SequenceNode {
	s := new(SequenceNode)
	s.compositeNodeBase = newCompositeNode(s)
	return s
}

func (s *SequenceNode) nodeType() nodeType { return ntSequence }

func (s *SequenceNode) createTask(parent task) task {
	return sequenceTaskPool.get().(*sequenceTask).ctr(s, parent)
}

func (s *SequenceNode) destroyTask(t task) {
	t.(*sequenceTask).dtr()
	sequenceTaskPool.put(t)
}

var sequenceTaskPool = newTaskPool(func() task { return newSequenceTask() })

type sequenceTask struct {
	*childSeqTask
}

func newSequenceTask() *sequenceTask {
	t := new(sequenceTask)
	t.childSeqTask = newChildSeqTask(t, sequenceKeepOn)
	return t
}

func (t *sequenceTask) ctr(node *SequenceNode, parent task) task {
	return t.childSeqTask.ctr(node, parent)
}

// -----------------------------------------------------------
// Selector
// -----------------------------------------------------------

func selectorKeepOn(r Result) bool {
	return r == RFailure
}

type SelectorNode struct {
	compositeNodeBase
}

func NewSelector() *SelectorNode {
	s := new(SelectorNode)
	s.compositeNodeBase = newCompositeNode(s)
	return s
}

func (s *SelectorNode) nodeType() nodeType { return ntSelector }

func (s *SelectorNode) createTask(parent task) task {
	return selectorTaskPool.get().(*selectorTask).ctr(s, parent)
}

func (s *SelectorNode) destroyTask(t task) {
	t.(*selectorTask).dtr()
	selectorTaskPool.put(t)
}

var selectorTaskPool = newTaskPool(func() task { return newSelectorTask() })

type selectorTask struct {
	*childSeqTask
}

func newSelectorTask() *selectorTask {
	t := new(selectorTask)
	t.childSeqTask = newChildSeqTask(t, selectorKeepOn)
	return t
}

func (t *selectorTask) ctr(node *SelectorNode, parent task) task {
	return t.childSeqTask.ctr(node, parent)
}

// -----------------------------------------------------------
// Random Child Sequence
// -----------------------------------------------------------

type randChildSeqTask struct {
	customSeqTask
	nodes   []node
	curNode int
}

func newRandChildSeqTask(self compositeTask, keepOn func(Result) bool) *randChildSeqTask {
	t := new(randChildSeqTask)
	t.customSeqTask = newCustomSeqTask(self, keepOn, t.getNextNode)
	return t
}

func (t *randChildSeqTask) dtr() {
	t.nodes = nil
	t.curNode = -1
	t.customSeqTask.dtr()
}

func (t *randChildSeqTask) onInit(e *Env) bool {
	compNode := t.getNode()
	if compNode.ChildCount() == 0 {
		return false
	}

	t.nodes = make([]node, compNode.ChildCount())
	n := 0
	for node := compNode.FirstChild(); node != nil; node = node.NextSibling() {
		t.nodes[n] = node
		n++
	}

	for ; n > 1; n-- {
		k := rand.Intn(n)
		if k != n-1 {
			tmp := t.nodes[n-1]
			t.nodes[n-1] = t.nodes[k]
			t.nodes[k] = tmp
		}
	}

	t.curNode = -1

	return t.customSeqTask.onInit(e)
}

func (t *randChildSeqTask) onUpdate(e *Env) Result {
	return t.customSeqTask.onUpdate(e)
}

func (t *randChildSeqTask) onTerminate(e *Env) {
	t.customSeqTask.onTerminate(e)
	t.nodes = nil
	t.curNode = -1
}

func (t *randChildSeqTask) getNextNode() node {
	t.curNode++
	if t.curNode == len(t.nodes) {
		t.curNode = -1
		t.customSeqTask.curNode = nil
		return nil
	} else {
		t.customSeqTask.curNode = t.nodes[t.curNode]
		return t.nodes[t.curNode]
	}
}

// -----------------------------------------------------------
// RandomSequence
// -----------------------------------------------------------

type RandSequenceNode struct {
	compositeNodeBase
}

func NewRandSequence() *RandSequenceNode {
	s := new(RandSequenceNode)
	s.compositeNodeBase = newCompositeNode(s)
	return s
}

func (s *RandSequenceNode) nodeType() nodeType { return ntRandSequence }

func (s *RandSequenceNode) createTask(parent task) task {
	return randSeqTaskPool.get().(*randSequenceTask).ctr(s, parent)
}

func (s *RandSequenceNode) destroyTask(t task) {
	t.(*randSequenceTask).dtr()
	randSeqTaskPool.put(t)
}

var randSeqTaskPool = newTaskPool(func() task { return newRandSequenceTask() })

type randSequenceTask struct {
	*randChildSeqTask
}

func newRandSequenceTask() *randSequenceTask {
	s := new(randSequenceTask)
	s.randChildSeqTask = newRandChildSeqTask(s, sequenceKeepOn)
	return s
}

func (t *randSequenceTask) ctr(node *RandSequenceNode, parent task) task {
	return t.randChildSeqTask.ctr(node, parent)
}

// -----------------------------------------------------------
// RandomSelector
// -----------------------------------------------------------

type RandSelectorNode struct {
	compositeNodeBase
}

func NewRandSelector() *RandSelectorNode {
	s := new(RandSelectorNode)
	s.compositeNodeBase = newCompositeNode(s)
	return s
}

func (s *RandSelectorNode) nodeType() nodeType { return ntRandSelector }

func (s *RandSelectorNode) createTask(parent task) task {
	return randSelcTaskPool.get().(*randSelectorTask).ctr(s, parent)
}

func (s *RandSelectorNode) destroyTask(t task) {
	t.(*randSelectorTask).dtr()
	randSelcTaskPool.put(t)
}

var randSelcTaskPool = newTaskPool(func() task { return newRandSelectorTask() })

type randSelectorTask struct {
	*randChildSeqTask
}

func newRandSelectorTask() *randSelectorTask {
	s := new(randSelectorTask)
	s.randChildSeqTask = newRandChildSeqTask(s, selectorKeepOn)
	return s
}

func (t *randSelectorTask) ctr(node *RandSelectorNode, parent task) task {
	return t.randChildSeqTask.ctr(node, parent)
}

// -----------------------------------------------------------
// Parallel
// -----------------------------------------------------------

type ParallelNode struct {
	compositeNodeBase
}

func NewParallel() *ParallelNode {
	p := new(ParallelNode)
	p.compositeNodeBase = newCompositeNode(p)
	return p
}

func (p *ParallelNode) nodeType() nodeType { return ntParallel }

func (p *ParallelNode) createTask(parent task) task {
	return paralTaskPool.get().(*parallelTask).ctr(p, parent)
}

func (p *ParallelNode) destroyTask(t task) {
	t.(*parallelTask).dtr()
	paralTaskPool.put(t)
}

var paralTaskPool = newTaskPool(func() task { return newParellelTask() })

type parallelTask struct {
	compositeTaskBase
	childs    []task
	completed int
}

func newParellelTask() *parallelTask {
	t := new(parallelTask)
	t.compositeTaskBase = newCompisteTask(t)
	return t
}

func (t *parallelTask) ctr(node *ParallelNode, parent task) task {
	return t.compositeTaskBase.ctr(node, parent)
}

func (t *parallelTask) dtr() {
	for i, v := range t.childs {
		if v != nil {
			if debug {
				log.Printf("parallelTask.dtr() No.%d child not nil", i)
			}

			v.destroy()
		}
	}
	t.childs = nil

	t.completed = 0

	t.compositeTaskBase.dtr()
}

func (t *parallelTask) onInit(e *Env) bool {
	node := t.getNode()
	if node.ChildCount() == 0 {
		return false
	}

	t.childs = make([]task, node.ChildCount())
	for i, node := 0, node.FirstChild(); node != nil; i, node = i+1, node.NextSibling() {
		t.childs[i] = node.createTask(t)
		e.pushCurrentTask(t.childs[i])
	}

	return true
}

func (t *parallelTask) onUpdate(e *Env) Result {
	return RRunning
}

func (t *parallelTask) onTerminate(e *Env) {
	t.childs = nil
	t.completed = 0
}

func (t *parallelTask) onStop(e *Env) {
	for _, v := range t.childs {
		if v != nil {
			v.detachParent()
		}
	}
}

func (t *parallelTask) onLazyStop(e *Env) {
	for _, v := range t.childs {
		if v != nil {
			v.lazyStop(e)
		}
	}
}

func (t *parallelTask) onChildOver(child task, r Result, e *Env) Result {
	for i, v := range t.childs {
		if v == nil {
			continue
		}

		if r == RSuccess && child == v {
			t.childs[i] = nil
			break
		} else if r == RFailure && child != v {
			v.lazyStop(e)
		}
	}

	if t.isLazyStop() {
		return RRunning
	}

	t.completed++
	if r == RSuccess && t.completed < len(t.childs) {
		r = RRunning
	}

	return r
}
