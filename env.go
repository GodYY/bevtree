package bevtree

type Env struct {
	updateSeri         uint32
	nodeQue            *nodeQueue
	nodeUpdateBoundary *nodeQueueElem
	DataContext
	userData interface{}
}

func NewEnv(userData interface{}) *Env {
	e := &Env{
		nodeQue:     newNodeQueue(),
		DataContext: NewBlackboard(),
		userData:    userData,
	}
	e.nodeUpdateBoundary = e.nodeQue.pushBack(nil)

	return e
}

func (e *Env) DataCtx() DataContext { return e.DataContext }

func (e *Env) UserData() interface{} { return e.userData }

func (e *Env) getUpdateSeri() uint32 { return e.updateSeri }

func (e *Env) noNodes() bool {
	return e.nodeQue.empty() || (e.nodeQue.len() == 1 && e.nodeQue.front() == e.nodeUpdateBoundary)
}

func (e *Env) pushNode(node node, nextRounds ...bool) {
	nextRound := false
	if len(nextRounds) > 0 {
		nextRound = nextRounds[0]
	}

	elem := node.getQueElem()
	if elem != nil {
		if elem.q == e.nodeQue {
			if nextRound {
				e.nodeQue.moveToBack(elem)
			} else {
				e.nodeQue.moveBefore(elem, e.nodeUpdateBoundary)
			}
			return
		}
		elem.q.remove(elem)
	}

	if nextRound {
		elem = e.nodeQue.pushBack(node)
	} else {
		elem = e.nodeQue.insertBefore(node, e.nodeUpdateBoundary)
	}
	node.setQueElem(elem)
}

func (e *Env) pushCurrentNode(node node) {
	e.pushNode(node)
}

func (e *Env) popCurrentNode() node {
	if e.nodeQue.front() == e.nodeUpdateBoundary {
		e.nodeQue.moveToBack(e.nodeUpdateBoundary)
		return nil
	}

	node := e.nodeQue.popFrontNode()
	if node != nil {
		node.setQueElem(nil)
	}

	return node
}

func (e *Env) pushNextNode(node node) {
	e.pushNode(node, true)
}

func (e *Env) update() uint32 {
	e.nodeQue.moveToBack(e.nodeUpdateBoundary)
	e.updateSeri++
	return e.updateSeri
}

func (e *Env) reset() {
	e.updateSeri = 0
	e.nodeQue.clear(e.onNodeClear)
	e.nodeUpdateBoundary = e.nodeQue.pushBack(nil)
	e.Clear()
}

func (e *Env) onNodeClear(n node) {
	n.setQueElem(nil)
}
