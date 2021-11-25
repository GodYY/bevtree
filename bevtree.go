package bevtree

import (
	"fmt"
	"log"
	"reflect"

	"github.com/GodYY/gutils/assert"
)

type NodeType int8

// Node metadata.
type nodeMETA struct {
	// node name.
	name string

	// type value.
	typ NodeType

	// function of creating node.
	creator func() Node

	// task pool.
	taskPool *pool
}

func (meta *nodeMETA) createNode() Node { return meta.creator() }

func (meta *nodeMETA) createTask(node Node) Task {
	assert.Assert(node != nil, "node nil")
	task := meta.taskPool.get().(Task)
	task.OnCreate(node)
	return task
}

func (meta *nodeMETA) destroyTask(task Task) {
	task.OnDestroy()
	meta.taskPool.put(task)
}

var nodeName2META = map[string]*nodeMETA{}
var nodeType2META = map[NodeType]*nodeMETA{}

func getNodeMETAByType(t NodeType) *nodeMETA { return nodeType2META[t] }

func (t NodeType) String() string {
	return getNodeMETAByType(t).name
}

// Register one node type.
func RegisterNodeType(name string, nodeCreator func() Node, taskCreator func() Task) NodeType {
	assert.NotEqual(name, "", "empty node type name")
	assert.AssertF(nodeCreator != nil, "node type \"%s\" nodeCreator nil", name)
	assert.AssertF(taskCreator != nil, "node type \"%s\" taskCreator nil", name)
	assert.AssertF(nodeName2META[name] == nil, "node type \"%s\" registered", name)

	meta := &nodeMETA{
		name:     name,
		typ:      NodeType(len(nodeName2META)),
		creator:  nodeCreator,
		taskPool: newPool(func() interface{} { return taskCreator() }),
	}

	nodeName2META[name] = meta
	nodeType2META[meta.typ] = meta

	return meta.typ
}

var (
	root            = RegisterNodeType("root", func() Node { return newRootNode() }, func() Task { return &rootTask{} })
	inverter        = RegisterNodeType("inverter", func() Node { return NewInverterNode() }, func() Task { return &inverterTask{} })
	succeeder       = RegisterNodeType("succeeder", func() Node { return NewSucceederNode() }, func() Task { return &succeederTask{} })
	repeater        = RegisterNodeType("repeater", func() Node { return newRepeaterNode() }, func() Task { return &repeaterTask{} })
	repeatUntilFail = RegisterNodeType("repeatuntilfail", func() Node { return NewRepeatUntilFailNode(true) }, func() Task { return &repeatUntilFailTask{} })
	sequence        = RegisterNodeType("sequence", func() Node { return NewSequenceNode() }, func() Task { return &sequenceTask{} })
	selector        = RegisterNodeType("selector", func() Node { return NewSelectorNode() }, func() Task { return &selectorTask{} })
	randSequence    = RegisterNodeType("randsequence", func() Node { return NewRandSequenceNode() }, func() Task { return &randSequenceTask{} })
	randSelector    = RegisterNodeType("randselector", func() Node { return NewRandSelectorNode() }, func() Task { return &randSelectorTask{} })
	parallel        = RegisterNodeType("parallel", func() Node { return NewParallelNode() }, func() Task { return &parallelTask{} })
	behavior        = RegisterNodeType("behavior", func() Node { return newBevNode() }, func() Task { return &bevTask{} })
)

func checkNodeTypes() {
	for _, v := range nodeType2META {
		node := v.createNode()

		assert.AssertF(node != nil, "node type \"%s\" create nil node", v.name)
		assert.EqualF(node.NodeType(), v.typ, "node created of type \"%s\" has different type \"%s\"", v.name, node.NodeType().String())

		destroyAgent(createAgent(node))
	}
}

func init() {
	checkNodeTypes()
}

type Node interface {
	NodeType() NodeType
	Parent() Node
	SetParent(Node)
}

type node struct {
	parent Node
}

func newNode() node {
	return node{}
}

func (n *node) Parent() Node { return n.parent }

func (n *node) SetParent(parent Node) {
	n.parent = parent
}

type status int8

const (
	sNone = status(iota)
	sRunning
	sTerminated
	sStopped
	sDestroyed
)

var statusStrings = [...]string{
	sNone:       "none",
	sRunning:    "running",
	sTerminated: "terminated",
	sStopped:    "stopped",
	sDestroyed:  "destroyed",
}

func (s status) String() string { return statusStrings[s] }

type lazyStop int8

const (
	lzsNone = lazyStop(iota)
	lzsBeforeUpdate
	lzsAfterUpdate
)

var lazyStopStrings = [...]string{
	lzsNone:         "none",
	lzsBeforeUpdate: "before-Update",
	lzsAfterUpdate:  "after-Update",
}

func (l lazyStop) String() string { return lazyStopStrings[l] }

type Result int8

const (
	RSuccess = Result(iota)
	RFailure
	RRunning
)

var resultStrings = [...]string{
	RSuccess: "success",
	RFailure: "failure",
	RRunning: "running",
}

func (r Result) String() string { return resultStrings[r] }

type TaskType int8

const (
	TaskSingle = TaskType(iota)
	TaskSerial
	TaskParallel
)

var taskTypeStrings = [...]string{
	TaskSingle:   "single",
	TaskSerial:   "serial",
	TaskParallel: "parallel",
}

func (tt TaskType) String() string { return taskTypeStrings[tt] }

type Task interface {
	TaskType() TaskType
	OnCreate(node Node)
	OnDestroy()
	OnInit(nextNodes *NodeList, ctx *Context) bool
	OnUpdate(ctx *Context) Result
	OnTerminate(ctx *Context)
	OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result
}

type agent struct {
	node             Node
	task             Task
	parent           *agent
	firstChild       *agent
	prev, next       *agent
	latestUpdateSeri uint32
	st               status
	lzStop           lazyStop
	elem             *element
}

func (a *agent) onCreate(node Node, task Task) {
	a.node = node
	a.task = task
	a.latestUpdateSeri = 0
	a.st = sNone
	a.lzStop = lzsNone
}

func (a *agent) onDestroy() {
	a.node = nil
	a.task = nil
}

func (a *agent) isPersistent() bool { return a.task.TaskType() == TaskSingle }

func (a *agent) getNext() *agent {
	if a.parent != nil && a.next != a.parent.firstChild {
		return a.next
	}
	return nil
}

func (a *agent) getPrev() *agent {
	if a.parent != nil && a.prev != a.parent.firstChild {
		return a.prev
	}
	return nil
}

func (a *agent) addChild(child *agent) {
	assert.Assert(child != nil && child.parent == nil, "child nil or has parent")

	if a.firstChild == nil {
		child.prev = child
		child.next = child
		a.firstChild = child
	} else {
		child.prev = a.firstChild.prev
		child.next = a.firstChild
		child.prev.next = child
		child.next.prev = child
	}

	child.parent = a
}

func (a *agent) removeChild(child *agent) {
	assert.Assert(child != nil && child.parent == a, "child nil or parent not match")

	if child == a.firstChild && a.firstChild.next == a.firstChild {
		a.firstChild = nil
	} else {
		if child == a.firstChild {
			a.firstChild = a.firstChild.next
		}

		child.prev.next = child.next
		child.next.prev = child.prev
	}

	child.prev = nil
	child.next = nil
	child.parent = nil
}

func (a *agent) getParent() *agent         { return a.parent }
func (a *agent) getStatus() status         { return a.st }
func (a *agent) setStatus(st status)       { a.st = st }
func (a *agent) getLZStop() lazyStop       { return a.lzStop }
func (a *agent) setLZStop(lzStop lazyStop) { a.lzStop = lzStop }
func (a *agent) getElem() *element         { return a.elem }
func (a *agent) setElem(elem *element)     { a.elem = elem }

func (a *agent) update(ctx *Context) Result {
	if debug {
		log.Printf("agent nodetype:%v update %v %v", a.node.NodeType(), a.getStatus(), a.getLZStop())
	}

	st := a.getStatus()

	if debug {
		assert.NotEqualF(st, sDestroyed, "agent nodetype:%v already destroyed", a.node.NodeType())
	}

	// Update seri.
	a.latestUpdateSeri = ctx.getUpdateSeri()

	// lazy Stop before Update.
	lzStop := a.getLZStop()
	if lzStop == lzsBeforeUpdate {
		return a.doLazyStop(ctx)
	}

	// init.
	if st == sNone {
		if !a.task.OnInit(ctx.getNodeList(), ctx) {
			a.task.OnTerminate(ctx)
			a.setStatus(sTerminated)
			return RFailure
		}

		if debug {
			switch a.task.TaskType() {
			case TaskSingle:
				assert.AssertF(ctx.getNodeList().Len() == 0, "node type \"%s\" has children", a.node.NodeType().String())

			case TaskSerial:
				assert.AssertF(ctx.getNodeList().Len() == 1, "node type \"%s\" have no or more than one child", a.node.NodeType().String())

			case TaskParallel:
				assert.AssertF(ctx.getNodeList().Len() > 0, "node type \"%s\" have no children", a.node.NodeType().String())
			}
		}

		a.processNextChildren(ctx)
	}

	// Update.
	result := a.task.OnUpdate(ctx)

	// lazy Stop after Update
	if lzStop == lzsAfterUpdate {
		return a.doLazyStop(ctx)
	}

	if result == RRunning {
		a.setStatus(sRunning)
	} else {
		// terminate.
		a.task.OnTerminate(ctx)
		a.setStatus(sTerminated)
	}

	return result
}

func (a *agent) processNextChildren(ctx *Context) {
	nodeList := ctx.getNodeList()
	for nextChildNode := nodeList.pop(); nextChildNode != nil; nextChildNode = nodeList.pop() {
		childAgent := createAgent(nextChildNode)
		a.addChild(childAgent)
		ctx.pushCurrentAgent(childAgent)
	}
}

func (a *agent) stop(ctx *Context) {
	if a.getStatus() != sRunning {
		return
	}

	if debug {
		log.Printf("agent nodetype:%v stop", a.node.NodeType())
	}

	child := a.firstChild
	for child != nil {
		node := child
		child = child.getNext()
		a.removeChild(node)
	}

	a.task.OnTerminate(ctx)
	a.setStatus(sStopped)
	a.setLZStop(lzsNone)
}

func (a *agent) lazyStop(ctx *Context) {
	if debug {
		log.Printf("agent nodetype:%v lazyStop", a.node.NodeType())
	}

	st := a.getStatus()
	if st == sStopped || st == sTerminated || a.getLZStop() != lzsNone {
		return
	}

	if a.latestUpdateSeri != ctx.getUpdateSeri() {
		a.setLZStop(lzsAfterUpdate)
	} else {
		a.setLZStop(lzsBeforeUpdate)
	}

	if a.elem == nil || a.getLZStop() == lzsBeforeUpdate {
		ctx.pushCurrentAgent(a)
	}
}

func (a *agent) doLazyStop(ctx *Context) Result {
	a.lazyStopChildren(ctx)
	a.task.OnTerminate(ctx)
	a.setStatus(sStopped)
	a.setLZStop(lzsNone)
	return RFailure
}

func (a *agent) lazyStopChildren(ctx *Context) {
	child := a.firstChild
	for child != nil {
		child.lazyStop(ctx)
		node := child
		child = child.getNext()
		a.removeChild(node)
	}
}

func (a *agent) onChildTerminated(child *agent, result Result, ctx *Context) Result {
	if debug {
		log.Printf("agent nodetype:%v onChildTerminated %v", a.node.NodeType(), result)
		assert.Assert(a.task.TaskType() != TaskSingle, "shouldnt be singletask")
		assert.Assert(child.getParent() == a, "invalid child")
		assert.NotEqual(result, RRunning, "child terminated with running")
	}

	a.removeChild(child)

	if a.getStatus() != sRunning {
		return RFailure
	}

	if a.getLZStop() != lzsNone {
		return RRunning
	}

	if result = a.task.OnChildTerminated(result, ctx.getNodeList(), ctx); result == RRunning {
		if debug {
			switch a.task.TaskType() {
			case TaskSerial:
				assert.AssertF(ctx.getNodeList().Len() == 1, "node type \"%s\" has no or more than one next child", a.node.NodeType().String())

			case TaskParallel:
				assert.AssertF(ctx.getNodeList().Len() == 0, "node type \"%s\" has next children", a.node.NodeType().String())
			}
		}

		a.processNextChildren(ctx)
	} else {
		if debug {
			assert.AssertF(ctx.getNodeList().Len() == 0, "node type \"%s\" has next children on terminating.", a.node.NodeType())
		}

		a.lazyStopChildren(ctx)
		a.task.OnTerminate(ctx)
		a.setStatus(sTerminated)
		a.setLZStop(lzsNone)
	}

	return result
}

var agentPool = newPool(func() interface{} { return &agent{} })

func createAgent(node Node) *agent {
	nodeMETA := getNodeMETAByType(node.NodeType())
	if nodeMETA == nil {
		panic(fmt.Sprintf("node type %d meta not found, %s", node.NodeType(), reflect.TypeOf(node).Elem().Name()))
	}

	task := nodeMETA.createTask(node)
	switch task.TaskType() {
	case TaskSingle, TaskSerial, TaskParallel:
	default:
		panic(fmt.Sprintf("node type \"%s\" create invalid type %d task", node.NodeType().String(), task.TaskType()))
	}

	agent := agentPool.get().(*agent)
	agent.onCreate(node, task)

	return agent
}

func destroyAgent(agent *agent) {
	if debug {
		assert.AssertF(agent.getElem() == nil, "agent node type \"%s\" still in list on destroy", agent.node.NodeType().String())
	}

	node := agent.node
	nodeMETA := getNodeMETAByType(node.NodeType())
	if nodeMETA == nil {
		panic(fmt.Sprintf("node type %d meta not found, %s", node.NodeType(), reflect.TypeOf(node).Elem().Name()))
	}

	nodeMETA.destroyTask(agent.task)
	agent.onDestroy()
	agentPool.put(agent)
}

type rootNode struct {
	child Node
}

func newRootNode() *rootNode {
	return &rootNode{}
}

func (rootNode) NodeType() NodeType { return root }
func (rootNode) Parent() Node       { return nil }
func (rootNode) SetParent(Node)     {}
func (r *rootNode) Child() Node     { return r.child }

func (r *rootNode) SetChild(child Node) {
	assert.Assert(child == nil || child.Parent() == nil, "child already has parent")

	if r.child != nil {
		r.child.SetParent(nil)
		r.child = nil
	}

	if child != nil {
		child.SetParent(r)
		r.child = child
	}
}

type rootTask struct {
	node *rootNode
}

func (r *rootTask) TaskType() TaskType { return TaskSerial }
func (r *rootTask) OnCreate(node Node) { r.node = node.(*rootNode) }
func (r *rootTask) OnDestroy()         { r.node = nil }

func (r *rootTask) OnInit(nextNodes *NodeList, ctx *Context) bool {
	if r.node.Child() == nil {
		return false
	} else {
		nextNodes.Push(r.node.Child())
		return true
	}
}

func (r *rootTask) OnUpdate(ctx *Context) Result { return RRunning }
func (r *rootTask) OnTerminate(ctx *Context)     {}
func (r *rootTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	return result
}

type BevTree struct {
	root *rootNode
}

func NewBevTree() *BevTree {
	tree := &BevTree{
		root: newRootNode(),
	}
	return tree
}

func (t *BevTree) Root() *rootNode { return t.root }

func (t *BevTree) Clear() {
	t.root.SetChild(nil)
}

func (t *BevTree) Update(ctx *Context) Result {
	if ctx.noAgents() {
		ctx.pushCurrentAgent(createAgent(t.root))
	}

	ctx.update()

	result := RRunning
	for agent := ctx.popCurrentAgent(); agent != nil; agent = ctx.popCurrentAgent() {
		r := agent.update(ctx)
		st := agent.getStatus()
		if st == sStopped {
			destroyAgent(agent)
			continue
		}

		if st == sTerminated {
			terminated := true
			for agent.getParent() != nil {
				parent := agent.getParent()
				parentTerminated := parent.getStatus() != sRunning

				r = parent.onChildTerminated(agent, r, ctx)
				if parentTerminated || r == RRunning {
					terminated = false
					break
				}

				assert.Assert(parent.getElem() == nil, "parent is still in work list")

				destroyAgent(agent)
				agent = parent
			}

			destroyAgent(agent)

			if terminated {
				assert.Equal(result, RRunning, "Update terminated reapeatedly")
				assert.NotEqual(r, RRunning, "Update terminated with RRunning")

				result = r
			}
		} else if agent.isPersistent() {
			ctx.pushNextAgent(agent)
		}
	}

	assert.Assert(result == RRunning || ctx.noAgents(), "Update terminated but already has agents")

	return result
}

func (t *BevTree) Stop(ctx *Context) {
	ctx.reset()
}
