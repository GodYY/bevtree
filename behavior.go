package bevtree

import "fmt"

type Behavior interface {
	OnStart(*Env)
	OnUpdate(*Env) Result
	OnEnd(*Env)
	OnStop(*Env)
}

type BevNode struct {
	nodeBase
	bev Behavior
}

func NewBehavior(bev Behavior) *BevNode {
	if bev == nil {
		panic("nil behavior")
	}

	return &BevNode{
		bev: bev,
	}
}

func (BevNode) ChildCount() int           { return 0 }
func (BevNode) AddChild(node)             {}
func (BevNode) RemoveChild(node)          {}
func (BevNode) AddChildBefore(_, _ node)  {}
func (BevNode) AddChildAfter(_, _ node)   {}
func (BevNode) MoveChildBefore(_, _ node) {}
func (BevNode) MoveChildAfter(_, _ node)  {}
func (BevNode) FirstChild() node          { return nil }
func (BevNode) LastChild() node           { return nil }

func (BevNode) isBehavior() bool { return true }

func (n *BevNode) update(e *Env) Result {
	n.latestUpdateSeri = e.getUpdateSeri()

	if n.status != sRunning {
		n.bev.OnStart(e)
	}

	if n.lzStop == lzsBeforeUpdate {
		return n.doLazyStop(e)
	}

	result := n.bev.OnUpdate(e)

	if n.lzStop == lzsAfterUpdate {
		return n.doLazyStop(e)
	}

	if result == RRunning {
		n.status = sRunning
	} else {
		n.bev.OnEnd(e)
		n.status = sNone
	}

	return result

}

func (n *BevNode) stop(e *Env) {
	if n.status == sNone || n.status == sStopped {
		return
	}

	n.bev.OnStop(e)
	n.bev.OnEnd(e)
	n.status = sStopped
	n.lzStop = lzsNone
}

func (n *BevNode) doLazyStop(e *Env) Result {
	fmt.Println("behavior doLazyStop", n.lzStop)
	n.bev.OnStop(e)
	n.bev.OnEnd(e)
	n.status = sStopped
	n.lzStop = lzsNone
	return RFailure
}

func (n *BevNode) childOver(node, Result, *Env) Result {
	panic("should not be called")
}
