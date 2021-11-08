package bevtree

import "math/rand"

type compositeNode = logicNode

type compoiteNodeBase struct {
	logicNodeBase
	childCount int
	firstChild node
	lastChild  node
}

func newCompositeNode(self compositeNode) compoiteNodeBase {
	if self == nil {
		panic("nil self")
	}

	return compoiteNodeBase{
		logicNodeBase: newLogicNode(self),
	}
}

func (c *compoiteNodeBase) ChildCount() int { return c.childCount }

func (c *compoiteNodeBase) FirstChild() node { return c.firstChild }

func (c *compoiteNodeBase) LastChild() node { return c.lastChild }

func (c *compoiteNodeBase) AddChild(child node) {
	if child == nil || child.Parent() != nil {
		panic("invalid child")
	} else if c.lastChild == nil {
		c.lastChild = child
		c.firstChild = child
		child.setParent(c.self)
		c.childCount++
	} else {
		c.AddChildAfter(child, c.lastChild)
	}
}

func (c *compoiteNodeBase) AddChildBefore(child, mark node) {
	if mark.Parent() != c.self {
		panic("mark not child of parent")
	}

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

func (c *compoiteNodeBase) AddChildAfter(child, mark node) {
	if mark.Parent() != c.self {
		panic("mark not child of parent")
	}

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

func (c *compoiteNodeBase) MoveChildBefore(child, mark node) {
	if c.self != child.Parent() && c.self != mark.Parent() {
		panic("parent missmatch")
	}

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

func (c *compoiteNodeBase) MoveChildAfter(child, mark node) {
	if c.self != child.Parent() && c.self != mark.Parent() {
		panic("parent missmatch")
	}

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

func (c *compoiteNodeBase) RemoveChild(child node) {
	if child == nil || child.Parent() != c.self {
		panic("invalid child")
	}

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

type customSeqNode struct {
	compoiteNodeBase
	curChild     node
	keepOn       func(Result) bool
	getNextChild func() node
}

func newCustomSeqNode(self logicNode, keepOn func(Result) bool, getNextChild func() node) customSeqNode {
	if keepOn == nil {
		panic("nil keepOn")
	}

	if getNextChild == nil {
		panic("nil getNextChild")
	}

	c := customSeqNode{
		compoiteNodeBase: newCompositeNode(self),
		keepOn:           keepOn,
		getNextChild:     getNextChild,
	}

	return c
}

func (c *customSeqNode) checkCompletedChild(child node) {
	if child != c.curChild {
		panic("not current child")
	}
}

func (c *customSeqNode) onStart(e *Env) {
	if c.ChildCount() > 0 {
		child := c.getNextChild()
		e.pushCurrentNode(child)
		c.curChild = child
	}
}

func (c *customSeqNode) onUpdate(e *Env) Result {
	if c.ChildCount() == 0 {
		return RFailure
	}

	return RRunning
}

func (c *customSeqNode) onLazyStop(e *Env) {
	if c.curChild != nil {
		c.childLazyStop(c.curChild, e)
	}
}

func (c *customSeqNode) onChildOver(child node, result Result, e *Env) Result {
	c.checkCompletedChild(child)

	if !c.keepOn(result) {
		return result
	}

	nextChild := c.getNextChild()
	if nextChild == nil {
		c.curChild = nil
		return result
	}

	e.pushNextNode(nextChild)
	c.curChild = nextChild
	return RRunning
}

func (c *customSeqNode) onEnd(e *Env) {
	c.curChild = nil
}

type childSeqNode struct {
	customSeqNode
}

func newChildSeqNode(self logicNode, keepOn func(Result) bool) *childSeqNode {
	c := new(childSeqNode)
	c.customSeqNode = newCustomSeqNode(self, keepOn, c.getNextChild)
	return c
}

func (c *childSeqNode) getNextChild() node {
	if c.curChild == nil {
		return c.firstChild
	} else {
		return c.curChild.NextSibling()
	}
}

func sequenceKeepOn(r Result) bool {
	return r == RSuccess
}

type SequenceNode struct {
	*childSeqNode
}

func NewSequence() *SequenceNode {
	s := new(SequenceNode)
	s.childSeqNode = newChildSeqNode(s, sequenceKeepOn)
	return s
}

func selectorKeepOn(r Result) bool {
	return r == RFailure
}

type SelectorNode struct {
	*childSeqNode
}

func NewSelector() *SelectorNode {
	s := new(SelectorNode)
	s.childSeqNode = newChildSeqNode(s, selectorKeepOn)
	return s
}

type randChildSeqNode struct {
	customSeqNode
	childs   []node
	curChild int
}

func newRandChildSeqNode(self logicNode, keepOn func(Result) bool) *randChildSeqNode {
	r := &randChildSeqNode{
		curChild: -1,
	}
	r.customSeqNode = newCustomSeqNode(self, keepOn, r.getNextChild)

	return r
}

func (r *randChildSeqNode) onStart(e *Env) {
	if r.ChildCount() == 0 {
		return
	}

	r.childs = make([]node, r.ChildCount())
	n := 0
	for node := r.FirstChild(); node != nil; node = node.NextSibling() {
		r.childs[n] = node
		n++
	}

	for n := len(r.childs); n > 1; n-- {
		k := rand.Intn(n)
		if k != n-1 {
			tmp := r.childs[n-1]
			r.childs[n-1] = r.childs[k]
			r.childs[k] = tmp
		}
	}

	r.customSeqNode.onStart(e)
}

func (r *randChildSeqNode) onEnd(e *Env) {
	r.customSeqNode.onEnd(e)
	r.childs = nil
	r.curChild = -1
}

func (r *randChildSeqNode) getNextChild() node {
	r.curChild++
	if r.curChild == len(r.childs) {
		r.curChild = -1
		r.customSeqNode.curChild = nil
		return nil
	} else {
		r.customSeqNode.curChild = r.childs[r.curChild]
		return r.childs[r.curChild]
	}
}

type RandSequenceNode struct {
	*randChildSeqNode
}

func NewRandSequence() *RandSequenceNode {
	s := new(RandSequenceNode)
	s.randChildSeqNode = newRandChildSeqNode(s, sequenceKeepOn)
	return s
}

type RandSelectorNode struct {
	*randChildSeqNode
}

func NewRandSelector() *RandSelectorNode {
	s := new(RandSelectorNode)
	s.randChildSeqNode = newRandChildSeqNode(s, selectorKeepOn)
	return s
}

type ParallelNode struct {
	compoiteNodeBase
	completed int
}

func NewParallel() *ParallelNode {
	p := new(ParallelNode)
	p.compoiteNodeBase = newCompositeNode(p)
	return p
}

func (p *ParallelNode) onStart(e *Env) {
	for node := p.FirstChild(); node != nil; node = node.NextSibling() {
		e.pushCurrentNode(node)
	}
}

func (p *ParallelNode) onUpdate(e *Env) Result {
	if p.ChildCount() == 0 {
		return RFailure
	}

	if p.completed < p.ChildCount() {
		return RRunning
	}

	return RSuccess
}

func (p *ParallelNode) onLazyStop(e *Env) {
	for node := p.FirstChild(); node != nil; node = node.NextSibling() {
		p.childLazyStop(node, e)
	}
}

func (p *ParallelNode) onChildOver(child node, result Result, e *Env) Result {
	p.completed++

	if result == RFailure {
		for node := p.FirstChild(); node != nil; node = node.NextSibling() {
			if node != child {
				p.childLazyStop(node, e)
			}
		}
	} else if p.completed < p.childCount {
		result = RRunning
	}

	return result
}

func (p *ParallelNode) onEnd(e *Env) {
	p.completed = 0
}
