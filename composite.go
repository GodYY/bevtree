package bevtree

import (
	"math/rand"
)

// The CompositeNode Interface represents the functions that
// the composite nodes in behavior tree must implement.
type CompositeNode interface {
	Node

	// Get the number of child nodes.
	ChildCount() int

	// Get the child node with index idx.
	Child(idx int) Node

	// Add a child node.
	AddChild(child Node)

	// Remove a child node with index idx.
	RemoveChild(idx int) Node
}

// The common part of composite node.
type compositeNode struct {
	node

	// The child nodes.
	children []Node
}

func newCompositeNode() compositeNode {
	return compositeNode{}
}

func (c *compositeNode) ChildCount() int { return len(c.children) }

func (c *compositeNode) Child(idx int) Node {
	if idx < 0 || idx >= c.ChildCount() {
		return nil
	}

	return c.children[idx]
}

func (c *compositeNode) addChild(child Node) bool {
	if child == nil || child.Parent() != nil {
		return false
	}

	c.children = append(c.children, child)
	return true
}

func (c *compositeNode) RemoveChild(idx int) Node {
	if idx < 0 || idx >= c.ChildCount() {
		return nil
	}

	child := c.children[idx]
	child.SetParent(nil)
	c.children = append(c.children[:idx], c.children[idx+1:]...)
	return child
}

// Sequence node runs child node one bye one until a child
// returns failure. It returns the result of the last
// running node.
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

// The sequence node task.
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

func (s *sequenceTask) OnInit(childNodes *NodeList, ctx *Context) bool {
	if s.node.ChildCount() == 0 {
		return false
	}

	childNodes.Push(s.node.Child(0))
	return true
}

func (s *sequenceTask) OnUpdate(ctx *Context) Result { return Running }
func (s *sequenceTask) OnTerminate(ctx *Context)     {}

func (s *sequenceTask) OnChildTerminated(result Result, childNodes *NodeList, ctx *Context) Result {
	s.curChildIdx++
	if result == Success && s.curChildIdx < s.node.ChildCount() {
		childNodes.Push(s.node.Child(s.curChildIdx))
		return Running
	} else {
		return result
	}
}

// Selector node runs child node one by one until a child
// returns success. It returns the result of the last
// running node.
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

// The selector node task.
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

func (s *selectorTask) OnInit(childNodes *NodeList, ctx *Context) bool {
	if s.node.ChildCount() == 0 {
		return false
	} else {
		childNodes.Push(s.node.Child(0))
		return true
	}
}

func (s *selectorTask) OnUpdate(ctx *Context) Result { return Running }
func (s *selectorTask) OnTerminate(ctx *Context)     {}

func (s *selectorTask) OnChildTerminated(result Result, childNodes *NodeList, ctx *Context) Result {
	s.curChildIdx++
	if result == Failure && s.curChildIdx < s.node.ChildCount() {
		childNodes.Push(s.node.Child(s.curChildIdx))
		return Running
	} else {
		return result
	}
}

// Get a random sequence of nodes.
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

// Random sequence runs child nodes one by one in a
// random sequence until a child returns failure. It
// returns the result of the last running node.
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

// The randome sequence node task.
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

func (s *randSequenceTask) OnInit(childNodes *NodeList, ctx *Context) bool {
	if s.childs = genRandNodes(s.node.children); len(s.childs) == 0 {
		return false
	} else {
		childNodes.Push(s.childs[s.curChildIdx])
		return true
	}
}

func (s *randSequenceTask) OnUpdate(ctx *Context) Result { return Running }
func (s *randSequenceTask) OnTerminate(ctx *Context)     {}

func (s *randSequenceTask) OnChildTerminated(result Result, childNodes *NodeList, ctx *Context) Result {
	s.curChildIdx++

	if result == Success && s.curChildIdx < s.node.ChildCount() {
		childNodes.Push(s.childs[s.curChildIdx])
		return Running
	} else {
		return result
	}
}

// Random selector node runs child nodes one by one in a
// random sequence until a child returns success. It returns
// the result of the last running node.
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

// The random selector task.
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

func (s *randSelectorTask) OnInit(childNodes *NodeList, ctx *Context) bool {
	s.childs = genRandNodes(s.node.children)
	if len(s.childs) == 0 {
		return false
	} else {
		childNodes.Push(s.childs[s.curChildIdx])
		return true
	}
}

func (s *randSelectorTask) OnUpdate(ctx *Context) Result { return Running }
func (s *randSelectorTask) OnTerminate(ctx *Context)     {}

func (s *randSelectorTask) OnChildTerminated(result Result, childNodes *NodeList, ctx *Context) Result {
	s.curChildIdx++

	if result == Failure && s.curChildIdx < s.node.ChildCount() {
		childNodes.Push(s.childs[s.curChildIdx])
		return Running
	} else {
		return result
	}
}

// The parrallel node runs child nodes together until a
// child returns failure. It returns success if all child
// nodes return success, or returns failure.
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

// The parallel node task.
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

func (p *parallelTask) OnInit(childNodes *NodeList, ctx *Context) bool {
	childCount := p.node.ChildCount()
	if childCount == 0 {
		return false
	} else {
		for i := 0; i < childCount; i++ {
			childNodes.Push(p.node.Child(i))
		}
		return true
	}
}

func (p *parallelTask) OnUpdate(ctx *Context) Result { return Running }
func (p *parallelTask) OnTerminate(ctx *Context)     { p.completed = 0 }
func (p *parallelTask) OnChildTerminated(result Result, childNodes *NodeList, ctx *Context) Result {
	p.completed++

	if result == Success && p.completed < p.node.ChildCount() {
		return Running
	} else {
		return result
	}
}
