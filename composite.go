package bevtree

import (
	"math/rand"
)

type CompositeNode interface {
	Node
	ChildCount() int
	Child(idx int) Node
	AddChild(child Node)
	RemoveChild(idx int) Node
}

type compositeNode struct {
	node
	childs []Node
}

func newCompositeNode() compositeNode {
	return compositeNode{}
}

func (c *compositeNode) ChildCount() int { return len(c.childs) }

func (c *compositeNode) Child(idx int) Node {
	if idx < 0 || idx >= c.ChildCount() {
		return nil
	}

	return c.childs[idx]
}

func (c *compositeNode) addChild(child Node) bool {
	if child == nil || child.Parent() != nil {
		return false
	}

	c.childs = append(c.childs, child)
	return true
}

func (c *compositeNode) RemoveChild(idx int) Node {
	if idx < 0 || idx >= c.ChildCount() {
		return nil
	}

	child := c.childs[idx]
	child.SetParent(nil)
	c.childs = append(c.childs[:idx], c.childs[idx+1:]...)
	return child
}

type SequenceNode struct {
	compositeNode
}

func NewSequenceNode() *SequenceNode {
	return &SequenceNode{
		compositeNode: newCompositeNode(),
	}
}

func (s *SequenceNode) NodeType() NodeType { return sequence }

func (s *SequenceNode) AddChild(child Node) {
	if s.compositeNode.addChild(child) {
		child.SetParent(s)
	}
}

type sequenceTask struct {
	node        *SequenceNode
	curChildIdx int
}

func (s *sequenceTask) TaskType() TaskType { return Serial }

func (s *sequenceTask) OnCreate(node Node) {
	s.node = node.(*SequenceNode)
	s.curChildIdx = 0
}

func (s *sequenceTask) OnDestroy() { s.node = nil }

func (s *sequenceTask) OnInit(nextList *NodeList, ctx *Context) bool {
	if s.node.ChildCount() == 0 {
		return false
	}

	nextList.Push(s.node.Child(0))
	return true
}

func (s *sequenceTask) OnUpdate(ctx *Context) Result { return Running }
func (s *sequenceTask) OnTerminate(ctx *Context)     {}

func (s *sequenceTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	s.curChildIdx++
	if result == Success && s.curChildIdx < s.node.ChildCount() {
		nextNodes.Push(s.node.Child(s.curChildIdx))
		return Running
	} else {
		return result
	}
}

type SelectorNode struct {
	compositeNode
}

func NewSelectorNode() *SelectorNode {
	return &SelectorNode{
		compositeNode: newCompositeNode(),
	}
}

func (s *SelectorNode) NodeType() NodeType { return selector }

func (s *SelectorNode) AddChild(child Node) {
	if s.compositeNode.addChild(child) {
		child.SetParent(s)
	}
}

type selectorTask struct {
	node        *SelectorNode
	curChildIdx int
}

func (s *selectorTask) TaskType() TaskType { return Serial }

func (s *selectorTask) OnCreate(node Node) {
	s.node = node.(*SelectorNode)
	s.curChildIdx = 0
}

func (s *selectorTask) OnDestroy() { s.node = nil }

func (s *selectorTask) OnInit(nextNodes *NodeList, ctx *Context) bool {
	if s.node.ChildCount() == 0 {
		return false
	} else {
		nextNodes.Push(s.node.Child(0))
		return true
	}
}

func (s *selectorTask) OnUpdate(ctx *Context) Result { return Running }
func (s *selectorTask) OnTerminate(ctx *Context)     {}

func (s *selectorTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	s.curChildIdx++
	if result == Failure && s.curChildIdx < s.node.ChildCount() {
		nextNodes.Push(s.node.Child(s.curChildIdx))
		return Running
	} else {
		return result
	}
}

func genRandNodes(nodes []Node) []Node {
	count := len(nodes)
	if count == 0 {
		return nil
	}

	result := make([]Node, count)
	for i := count - 1; i > 0; i-- {
		if result[i] == nil {
			result[i] = nodes[i]
		}

		k := rand.Intn(i + 1)
		if k != i {
			if result[k] == nil {
				result[k] = nodes[k]
			}

			result[k], result[i] = result[i], result[k]
		}
	}

	if result[0] == nil {
		result[0] = nodes[0]
	}

	return result
}

type RandSequenceNode struct {
	compositeNode
}

func NewRandSequenceNode() *RandSequenceNode {
	return &RandSequenceNode{
		compositeNode: newCompositeNode(),
	}
}

func (s *RandSequenceNode) NodeType() NodeType { return randSequence }

func (s *RandSequenceNode) AddChild(child Node) {
	if s.compositeNode.addChild(child) {
		child.SetParent(s)
	}
}

type randSequenceTask struct {
	node        *RandSequenceNode
	childs      []Node
	curChildIdx int
}

func (s *randSequenceTask) TaskType() TaskType { return Serial }

func (s *randSequenceTask) OnCreate(node Node) {
	s.node = node.(*RandSequenceNode)
	s.curChildIdx = 0
}

func (s *randSequenceTask) OnDestroy() {
	s.node = nil
	s.childs = nil
}

func (s *randSequenceTask) OnInit(nextNodes *NodeList, ctx *Context) bool {
	if s.childs = genRandNodes(s.node.childs); len(s.childs) == 0 {
		return false
	} else {
		nextNodes.Push(s.childs[s.curChildIdx])
		return true
	}
}

func (s *randSequenceTask) OnUpdate(ctx *Context) Result { return Running }
func (s *randSequenceTask) OnTerminate(ctx *Context)     {}

func (s *randSequenceTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	s.curChildIdx++

	if result == Success && s.curChildIdx < s.node.ChildCount() {
		nextNodes.Push(s.childs[s.curChildIdx])
		return Running
	} else {
		return result
	}
}

type RandSelectorNode struct {
	compositeNode
}

func NewRandSelectorNode() *RandSelectorNode {
	return &RandSelectorNode{
		compositeNode: newCompositeNode(),
	}
}

func (s *RandSelectorNode) NodeType() NodeType { return randSelector }

func (s *RandSelectorNode) AddChild(child Node) {
	if s.compositeNode.addChild(child) {
		child.SetParent(s)
	}
}

type randSelectorTask struct {
	node        *RandSelectorNode
	childs      []Node
	curChildIdx int
}

func (s *randSelectorTask) TaskType() TaskType { return Serial }

func (s *randSelectorTask) OnCreate(node Node) {
	s.node = node.(*RandSelectorNode)
	s.curChildIdx = 0
}

func (s *randSelectorTask) OnDestroy() {
	s.node = nil
	s.childs = nil
}

func (s *randSelectorTask) OnInit(nextNodes *NodeList, ctx *Context) bool {
	s.childs = genRandNodes(s.node.childs)
	if len(s.childs) == 0 {
		return false
	} else {
		nextNodes.Push(s.childs[s.curChildIdx])
		return true
	}
}

func (s *randSelectorTask) OnUpdate(ctx *Context) Result { return Running }
func (s *randSelectorTask) OnTerminate(ctx *Context)     {}

func (s *randSelectorTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	s.curChildIdx++

	if result == Failure && s.curChildIdx < s.node.ChildCount() {
		nextNodes.Push(s.childs[s.curChildIdx])
		return Running
	} else {
		return result
	}
}

type ParallelNode struct {
	compositeNode
}

func NewParallelNode() *ParallelNode {
	return &ParallelNode{
		compositeNode: newCompositeNode(),
	}
}

func (p *ParallelNode) NodeType() NodeType { return parallel }

func (p *ParallelNode) AddChild(child Node) {
	if p.compositeNode.addChild(child) {
		child.SetParent(p)
	}
}

type parallelTask struct {
	node      *ParallelNode
	completed int
}

func (p *parallelTask) TaskType() TaskType { return Parallel }

func (p *parallelTask) OnCreate(node Node) {
	p.node = node.(*ParallelNode)
	p.completed = 0
}

func (p *parallelTask) OnDestroy() { p.node = nil }

func (p *parallelTask) OnInit(nextNodes *NodeList, ctx *Context) bool {
	childCount := p.node.ChildCount()
	if childCount == 0 {
		return false
	} else {
		for i := 0; i < childCount; i++ {
			nextNodes.Push(p.node.Child(i))
		}
		return true
	}
}

func (p *parallelTask) OnUpdate(ctx *Context) Result { return Running }
func (p *parallelTask) OnTerminate(ctx *Context)     { p.completed = 0 }
func (p *parallelTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	p.completed++

	if result == Success && p.completed < p.node.ChildCount() {
		return Running
	} else {
		return result
	}
}
