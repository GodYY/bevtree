package bevtree

import (
	"log"
	"math/rand"
	"reflect"
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
		assertNilArg(self, "self")
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
	assert(child != nil && child.Parent() == nil, "child nil or already has parent")

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
	assert(child != nil && child.Parent() == nil, "child nil or already has parent")
	assert(mark != nil && mark.Parent() == c.self, "mark nil or not child of it")

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
	assert(child != nil && child.Parent() == nil, "child nil or already has parent")
	assert(mark != nil && mark.Parent() == c.self, "mark nil or not child of it")

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
	assert(child != nil && child.Parent() == c.self, "child nil or not child of it")
	assert(mark != nil && mark.Parent() == c.self, "mark nil or not child of it")

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
	assert(child != nil && child.Parent() == c.self, "child nil or not child of it")
	assert(mark != nil && mark.Parent() == c.self, "mark nil or not child of it")

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
	assert(child != nil && child.Parent() == c.self, "child nil or not child of it")

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

func newCompisteTask(self compositeTask, node node, parent task) compositeTaskBase {
	return compositeTaskBase{
		logicTaskBase: newLogicTask(self, node, parent),
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

func newCustomSeqTask(self compositeTask, node node, parent task, keepOn func(Result) bool, getNextNode func() node) customSeqTask {
	if debug {
		assertNilArg(keepOn, "keepOn")
		assertNilArg(getNextNode, "getNextChild")
	}

	return customSeqTask{
		compositeTaskBase: newCompisteTask(self, node, parent),
		keepOn:            keepOn,
		getNextNode:       getNextNode,
	}
}

func (t *customSeqTask) checkChild(child task) {
	if child != t.curChild {
		panic("not current child")
	}
}

func (t *customSeqTask) onInit(e *Env) bool {
	if debug {
		log.Printf("%s.onInit", reflect.TypeOf(t.self).Elem().Name())
	}

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

func (t *customSeqTask) onLazyStop(e *Env) {
	t.childLazyStop(t.curChild, e)
}

func (t *customSeqTask) onChildOver(child task, r Result, e *Env) Result {
	assert(child == t.curChild, "not current child")

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

func newChildSeqTask(self compositeTask, node node, parent task, keepOn func(Result) bool) *childSeqTask {
	t := new(childSeqTask)
	t.customSeqTask = newCustomSeqTask(self, node, parent, keepOn, t.getNextNode)
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

func (s *SequenceNode) createTask(parent task) task {
	return newSequenceTask(s, parent)
}

func (s *SequenceNode) destroyTask(t task) {}

type sequenceTask struct {
	*childSeqTask
}

func newSequenceTask(node node, parent task) *sequenceTask {
	t := new(sequenceTask)
	t.childSeqTask = newChildSeqTask(t, node, parent, sequenceKeepOn)
	return t
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

func (s *SelectorNode) createTask(parent task) task {
	return newSelectorTask(s, parent)
}

func (s *SelectorNode) destroyTask(t task) {}

type selectorTask struct {
	*childSeqTask
}

func newSelectorTask(node node, parent task) *selectorTask {
	t := new(selectorTask)
	t.childSeqTask = newChildSeqTask(t, node, parent, selectorKeepOn)
	return t
}

// -----------------------------------------------------------
// Random Child Sequence
// -----------------------------------------------------------

type randChildSeqTask struct {
	customSeqTask
	nodes   []node
	curNode int
}

func newRandChildSeqTask(self compositeTask, node node, parent task, keepOn func(Result) bool) *randChildSeqTask {
	t := new(randChildSeqTask)
	t.customSeqTask = newCustomSeqTask(self, node, parent, keepOn, t.getNextNode)
	return t
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

func (s *RandSequenceNode) createTask(parent task) task {
	return newRandSequenceTask(s, parent)
}

func (s *RandSequenceNode) destroyTask(t task) {}

type randSequenceTask struct {
	*randChildSeqTask
}

func newRandSequenceTask(node node, parent task) *randSequenceTask {
	s := new(randSequenceTask)
	s.randChildSeqTask = newRandChildSeqTask(s, node, parent, sequenceKeepOn)
	return s
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

func (s *RandSelectorNode) createTask(parent task) task {
	return newRandSelectorTask(s, parent)
}

func (s *RandSelectorNode) destroyTask(t task) {}

type randSelectorTask struct {
	*randChildSeqTask
}

func newRandSelectorTask(node node, parent task) *randSelectorTask {
	s := new(randSelectorTask)
	s.randChildSeqTask = newRandChildSeqTask(s, node, parent, selectorKeepOn)
	return s
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

func (p *ParallelNode) createTask(parent task) task {
	return newParellelTask(p, parent)
}

func (p *ParallelNode) destroyTask(t task) {}

type parallelTask struct {
	compositeTaskBase
	childs    []task
	completed int
}

func newParellelTask(node node, parent task) *parallelTask {
	t := new(parallelTask)
	t.compositeTaskBase = newCompisteTask(t, node, parent)
	return t
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

func (t *parallelTask) onLazyStop(e *Env) {
	for _, v := range t.childs {
		t.childLazyStop(v, e)
	}
}

func (t *parallelTask) onChildOver(child task, r Result, e *Env) Result {
	t.completed++

	if r == RFailure {
		for _, v := range t.childs {
			if child != v {
				t.childLazyStop(v, e)
			}
		}
	} else if t.completed < len(t.childs) {
		r = RRunning
	}

	return r
}
