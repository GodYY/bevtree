package bevtree

import (
	"log"

	"github.com/GodYY/gutils/assert"
	"github.com/GodYY/gutils/finalize"
)

type NodeList struct {
	l list
}

func newNodeList() *NodeList {
	return &NodeList{}
}

func (nl *NodeList) Push(node Node) {
	assert.Assert(node != nil, "node nil")
	nl.l.pushBack(node)
}

func (nl *NodeList) pop() Node {
	if elem := nl.l.front(); elem != nil {
		return nl.l.remove(elem).(Node)
	} else {
		return nil
	}
}

func (nl *NodeList) Len() int { return nl.l.getLen() }

func (nl *NodeList) clear() { nl.l.init() }

type Context struct {
	updateSeri          uint32
	agentList           *list
	agentUpdateBoundary *element
	nodeList            *NodeList
	*dataSet
	userData interface{}
}

func NewContext(userData interface{}) *Context {
	ctx := &Context{
		agentList: newList(),
		nodeList:  newNodeList(),
		dataSet:   newDataSet(),
		userData:  userData,
	}

	finalize.SetFinalizer(ctx)

	return ctx
}

func (ctx *Context) Release() {
	finalize.UnsetFinalizer(ctx)
	ctx.release()
}

func (ctx *Context) release() {
	ctx.clearAgent()
	ctx.nodeList.clear()
	ctx.dataSet.clear()
	ctx.dataSet = nil
	ctx.userData = nil
}

func (ctx *Context) Finalizer() {
	if debug {
		log.Println("Env.Finalizer")
	}
	ctx.release()
}

func (ctx *Context) reset() {
	ctx.updateSeri = 0
	ctx.clearAgent()
	ctx.agentUpdateBoundary = nil
	ctx.dataSet.clear()
}

func (ctx *Context) UserData() interface{} { return ctx.userData }

func (ctx *Context) getUpdateSeri() uint32 { return ctx.updateSeri }

func (ctx *Context) noAgents() bool {
	return ctx.agentList.getLen() == 0 || (ctx.agentList.getLen() == 1 && ctx.agentList.front() == ctx.agentUpdateBoundary)
}

func (ctx *Context) getNodeList() *NodeList { return ctx.nodeList }

func (ctx *Context) lazyPushUpdateBoundary() {
	if ctx.agentUpdateBoundary == nil {
		ctx.agentUpdateBoundary = ctx.agentList.pushBack(nil)
	}
}

func (ctx *Context) pushAgent(agent *agent, nextRound bool) {
	assert.Assert(agent != nil, "agent nil")

	ctx.lazyPushUpdateBoundary()

	elem := agent.getElem()

	if elem == nil {
		if nextRound {
			elem = ctx.agentList.pushBack(agent)
		} else {
			elem = ctx.agentList.insertBefore(agent, ctx.agentUpdateBoundary)
		}

		agent.setElem(elem)
	} else {
		if nextRound {
			ctx.agentList.moveToBack(elem)
		} else {
			ctx.agentList.moveBefore(elem, ctx.agentUpdateBoundary)
		}
	}
}

func (ctx *Context) pushCurrentAgent(agent *agent) {
	ctx.pushAgent(agent, false)
}

func (ctx *Context) popCurrentAgent() *agent {
	ctx.lazyPushUpdateBoundary()

	elem := ctx.agentList.front()
	if elem == ctx.agentUpdateBoundary {
		ctx.agentList.moveToBack(elem)
		return nil
	}

	agent := elem.Value.(*agent)
	agent.setElem(nil)
	ctx.agentList.remove(elem)

	return agent
}

func (ctx *Context) pushNextAgent(agent *agent) {
	ctx.pushAgent(agent, true)
}

func (ctx *Context) removeAgent(agent *agent) {
	elem := agent.getElem()
	if elem != nil {
		ctx.agentList.remove(elem)
		agent.setElem(nil)
	}
}

func (ctx *Context) clearAgent() {
	elem := ctx.agentList.front()
	for elem != nil {
		next := elem.getNext()
		agent, ok := ctx.agentList.remove(elem).(*agent)
		elem = next

		if ok && agent != nil {
			assert.Assert(agent.isPersistent(), "agent is not persistent")

			for agent != nil {
				agent.setElem(nil)
				parent := agent.getParent()
				agent.stop(ctx)
				destroyAgent(agent)
				agent = parent
			}
		}
	}
}

func (ctx *Context) update() uint32 {
	ctx.lazyPushUpdateBoundary()
	ctx.updateSeri++
	return ctx.updateSeri
}
