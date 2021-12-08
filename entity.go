package bevtree

import (
	"log"

	"github.com/GodYY/gutils/assert"
	"github.com/GodYY/gutils/finalize"
)

// NodeList interface.
type NodeList interface {
	// Push one node.
	PushNode(Node)

	// Push nodes.
	PushNodes(...Node)
}

type nodeList struct {
	l list
}

func newNodeList() *nodeList {
	return &nodeList{}
}

func (nl *nodeList) PushNode(node Node) {
	assert.Assert(node != nil, "node nil")
	nl.l.pushBack(node)
}

func (nl *nodeList) PushNodes(nodes ...Node) {
	assert.Assert(len(nodes) > 0, "no nodes")
	for _, node := range nodes {
		nl.l.pushBack(node)
	}
}

func (nl *nodeList) pop() Node {
	if elem := nl.l.front(); elem != nil {
		return nl.l.remove(elem).(Node)
	} else {
		return nil
	}
}

func (nl *nodeList) len() int { return nl.l.getLen() }

func (nl *nodeList) clear() { nl.l.init() }

// Entity used to run a behavior tree.
type Entity interface {
	// Get the corresponding behavior tree.
	Tree() *Tree

	// Get the user data.
	UserData() interface{}

	// Get the context.
	Context() Context

	// Update behavior tree and get a result from this
	// update.
	Update() Result

	// Stops running the bahavior tree.
	Stop()

	// If the entity is no longer used, call Release to
	// release resource of it.
	Release()
}

// Entity implementation.
type entity struct {
	// Corresponding behavior tree.
	bevtree *Tree

	// The context.
	ctx *context

	// Agent list.
	agentList *list

	// Agent updating boundary. Agents behind the boundary
	// will update at next updateing.
	agentUpdateBoundary *element

	// Child node list. It is used to temporarily store
	// subsequent child nodes.
	childNodeList *nodeList
}

func NewEntity(bevtree *Tree, userData interface{}) Entity {
	return newEntity(bevtree, newContext(userData))
}

func newEntity(bevtree *Tree, ctx *context) *entity {
	assert.Assert(bevtree != nil, "bevtree nil")
	assert.Assert(ctx != nil, "ctx nil")

	entity := &entity{
		bevtree:       bevtree,
		ctx:           ctx,
		agentList:     newList(),
		childNodeList: newNodeList(),
	}

	finalize.SetFinalizer(entity)

	return entity
}

// If the entity is no longer used, call Release to
// release resource of it.
func (e *entity) Release() {
	finalize.UnsetFinalizer(e)
	e.release()
}

func (e *entity) release() {
	e.clearAgent()
	e.agentList = nil
	e.childNodeList.clear()
	e.childNodeList = nil
	e.ctx.release()
	e.ctx = nil
	e.bevtree = nil
	e.agentUpdateBoundary = nil
}

// Finalizer will be called by GC if there is no explicitly
// call Release.
func (e *entity) Finalizer() {
	if debug {
		log.Println("Env.Finalizer")
	}
	e.release()
}

func (e *entity) Tree() *Tree { return e.bevtree }

func (e *entity) Context() Context { return e.ctx }

func (e *entity) UserData() interface{} { return e.ctx.UserData() }

func (e *entity) getUpdateSeri() uint32 { return e.ctx.UpdateSeri() }

func (e *entity) getChildNodeList() *nodeList { return e.childNodeList }

func (e *entity) noAgents() bool {
	return e.agentList.getLen() == 0 || (e.agentList.getLen() == 1 && e.agentList.front() == e.agentUpdateBoundary)
}

func (e *entity) lazyPushUpdateBoundary() {
	if e.agentUpdateBoundary == nil {
		e.agentUpdateBoundary = e.agentList.pushBack(nil)
	}
}

func (e *entity) pushAgent_(agent *agent, nextRound bool) {
	assert.Assert(agent != nil, "agent nil")

	e.lazyPushUpdateBoundary()

	elem := agent.getElem()

	if elem == nil {
		if nextRound {
			elem = e.agentList.pushBack(agent)
		} else {
			elem = e.agentList.insertBefore(agent, e.agentUpdateBoundary)
		}

		agent.setElem(elem)
	} else {
		if nextRound {
			e.agentList.moveToBack(elem)
		} else {
			e.agentList.moveBefore(elem, e.agentUpdateBoundary)
		}
	}
}

// Push a agent that need to run in current update。
func (e *entity) pushAgent(agent *agent) {
	e.pushAgent_(agent, false)
}

// Pop a agent that need to run in current update.
func (e *entity) popAgent() *agent {
	e.lazyPushUpdateBoundary()

	elem := e.agentList.front()
	if elem == e.agentUpdateBoundary {
		e.agentList.moveToBack(elem)
		return nil
	}

	agent := elem.Value.(*agent)
	agent.setElem(nil)
	e.agentList.remove(elem)

	return agent
}

// Push a agent that need to run in the next update.
func (e *entity) pushPendingAgent(agent *agent) {
	e.pushAgent_(agent, true)
}

func (e *entity) removeAgent(agent *agent) {
	elem := agent.getElem()
	if elem != nil {
		e.agentList.remove(elem)
		agent.setElem(nil)
	}
}

func (e *entity) clearAgent() {
	elem := e.agentList.front()
	for elem != nil {
		next := elem.getNext()
		agent, ok := e.agentList.remove(elem).(*agent)
		elem = next

		if ok && agent != nil {
			assert.Assert(agent.isPersistent(), "agent is not persistent")

			for agent != nil {
				agent.setElem(nil)
				parent := agent.getParent()
				agent.stop(e.ctx)
				destroyAgent(agent)
				agent = parent
			}
		}
	}
}

// Update used to update the behavior tree and get a result
// from this update.
func (e *entity) Update() Result {
	e.lazyPushUpdateBoundary()
	e.ctx.update()

	if e.noAgents() {
		// No agents indicate the behavior tree was not run yet
		// or it had completed a traversal from root to root node.
		// Need to start a new traversal from the root node.
		e.pushAgent(createAgent(e.bevtree.Root()))
	}

	// The default result.
	result := Running

	// Run agent one by one until there are no agents at current
	// updating or back to root node.
	for agent := e.popAgent(); agent != nil; agent = e.popAgent() {
		r := agent.update(e)
		st := agent.getStatus()
		if st == sStopped {
			destroyAgent(agent)
			continue
		}

		if st == sTerminated {
			// agent terminated, submit result to parent for
			// making decision.

			// The flag indicating whether to back to the root
			// node.
			isBackToRoot := true

			// Submit result to parent until no parent.
			for agent.getParent() != nil {
				parent := agent.getParent()
				parentTerminated := parent.getStatus() != sRunning

				r = parent.onChildTerminated(agent, r, e)
				if parentTerminated || r == Running {
					// Parent already terminated or still running, stop.
					isBackToRoot = false
					break
				}

				assert.Assert(parent.getElem() == nil, "parent is still in work list")

				// Destroy the child agent.
				destroyAgent(agent)

				agent = parent
			}

			// Destroy the last terminated agent.
			destroyAgent(agent)

			if isBackToRoot {
				// Back to root node, update result.

				assert.Equal(result, Running, "Update terminated reapeatedly")
				assert.NotEqual(r, Running, "Update terminated with RRunning")

				result = r
			}
		} else if agent.isPersistent() {
			// agent still running and persistent, set it to
			// update at the next updating.

			e.pushPendingAgent(agent)
		}
	}

	assert.Assert(result == Running || e.noAgents(), "Update terminated but already has agents")

	return result
}

// Stop stops running the behavior tree.
func (e *entity) Stop() {
	e.ctx.reset()
	e.clearAgent()
	e.agentUpdateBoundary = nil
}
