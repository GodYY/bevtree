package bevtree

import (
	"github.com/GodYY/gutils/assert"
)

type SubtreeNode struct {
	node
	subtree            *Tree
	independentDataSet bool
}

func NewSubtreeNode(subtree *Tree, independentDataSet bool) *SubtreeNode {
	assert.Assert(subtree != nil, "subtree inil")
	return &SubtreeNode{
		node:               newNode(),
		subtree:            subtree,
		independentDataSet: independentDataSet,
	}
}

func (s *SubtreeNode) NodeType() NodeType { return subtree }

func (s *SubtreeNode) Subtree() *Tree { return s.subtree }

func (s *SubtreeNode) SetSubtree(subtree *Tree) { s.subtree = subtree }

func (s *SubtreeNode) IndependentDataSet() bool { return s.independentDataSet }

func (s *SubtreeNode) SetIndependentDataSet(independentDataSet bool) {
	s.independentDataSet = independentDataSet
}

type subtreeTask struct {
	node   *SubtreeNode
	entity Entity
}

// Get the TaskType.
func (s *subtreeTask) TaskType() TaskType { return Single }

// OnCreate is called immediately after the Task is created.
// node indicates the node on which the Task is created.
func (s *subtreeTask) OnCreate(node Node) { s.node = node.(*SubtreeNode) }

// OnDestroy is called before the Task is destroyed.
func (s *subtreeTask) OnDestroy() { s.node = nil }

// OnInit is called before the first update of the Task.
// nextChildNodes is used to return the child nodes that need
// to run next. ctx represents the running context of the
// behavior tree.
func (s *subtreeTask) OnInit(_ NodeList, ctx Context) bool {
	if s.node.Subtree() == nil {
		return false
	} else {
		s.entity = newEntity(s.node.Subtree(), ctx.(*context).clone(s.node.independentDataSet))
		return true
	}
}

// OnUpdate is called until the Task is terminated.
func (s *subtreeTask) OnUpdate(ctx Context) Result {
	return s.entity.Update()
}

// OnTerminate is called after ths last update of the Task.
func (s *subtreeTask) OnTerminate(ctx Context) {
	if s.entity != nil {
		s.entity.Release()
		s.entity = nil
	}
}

// OnChildTerminated is called when a sub Task is terminated.
//
// result Indicates the running result of the subtask.
// nextChildNodes is used to return the child nodes that need to
// run next.
//
// OnChildTerminated returns the decision result.
func (s *subtreeTask) OnChildTerminated(result Result, _ NodeList, _ Context) Result {
	panic("shouldnt be invoked")
}
