package bevtree

import (
	"fmt"
	"log"
	"reflect"

	"github.com/GodYY/gutils/assert"
)

// Node type.
type NodeType int8

// Node metadata.
type nodeMETA struct {
	// Node type name.
	name string

	// Node type value, assigned at registration.
	typ NodeType

	// The creator of node.
	creator func() Node

	// task pool, used to cache the destroyed task of this type of node.
	taskPool *pool
}

// Use creator to create node.
func (meta *nodeMETA) createNode() Node { return meta.creator() }

// Create a task of this type of node. First, get a cached task or
// create a new task. Then, call the OnCreate method of it.
func (meta *nodeMETA) createTask(node Node) Task {
	assert.Assert(node != nil, "node nil")
	task := meta.taskPool.get().(Task)
	task.OnCreate(node)
	return task
}

// Destroy a task of this type of node. First, call the OnDestroy
// method of the task. Then, put it to be cached to the pool.
func (meta *nodeMETA) destroyTask(task Task) {
	task.OnDestroy()
	meta.taskPool.put(task)
}

// The mapping of node type name to metadata.
var nodeName2META = map[string]*nodeMETA{}

// The mapping of node type to metadata.
var nodeType2META = map[NodeType]*nodeMETA{}

// Get metadata of node type t.
func getNodeMETAByType(t NodeType) *nodeMETA { return nodeType2META[t] }

func (t NodeType) String() string {
	return getNodeMETAByType(t).name
}

// Register a type of node. It create metadata of the type of node,
// and assigned it a type value.
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

// The default node types.
var (
	// The root node of behavior tree.
	root = RegisterNodeType("root", func() Node { return newRootNode() }, func() Task { return &rootTask{} })

	// The inverter node.
	inverter = RegisterNodeType("inverter", func() Node { return NewInverterNode() }, func() Task { return &inverterTask{} })

	// The succeeder node.
	succeeder = RegisterNodeType("succeeder", func() Node { return NewSucceederNode() }, func() Task { return &succeederTask{} })

	// The repeater node.
	repeater = RegisterNodeType("repeater", func() Node { return NewRepeaterNode(1) }, func() Task { return &repeaterTask{} })

	// The repeat-until-fail node.
	repeatUntilFail = RegisterNodeType("repeatuntilfail", func() Node { return NewRepeatUntilFailNode(false) }, func() Task { return &repeatUntilFailTask{} })

	// The sequence node.
	sequence = RegisterNodeType("sequence", func() Node { return NewSequenceNode() }, func() Task { return &sequenceTask{} })

	// The selector node.
	selector = RegisterNodeType("selector", func() Node { return NewSelectorNode() }, func() Task { return &selectorTask{} })

	// The random sequence node.
	randSequence = RegisterNodeType("randsequence", func() Node { return NewRandSequenceNode() }, func() Task { return &randSequenceTask{} })

	// The random selector node.
	randSelector = RegisterNodeType("randselector", func() Node { return NewRandSelectorNode() }, func() Task { return &randSelectorTask{} })

	// The parallel node.
	parallel = RegisterNodeType("parallel", func() Node { return NewParallelNode() }, func() Task { return &parallelTask{} })

	// The behavior node.
	behavior = RegisterNodeType("behavior", func() Node { return NewBevNode(nil) }, func() Task { return &bevTask{} })
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

// Node represents the structure portion of behavior tree.
// Node defines the basic function of node of behavior tree.
// Nodes must implement the Node interface to be considered
// behavior tree nodes.
type Node interface {
	// Node type.
	NodeType() NodeType

	// Get the parent node.
	Parent() Node

	// Set the parent node.
	SetParent(Node)

	// Get the comment.
	Comment() string

	// Set the comment.
	SetComment(string)
}

// The common part of node.
type node struct {
	parent  Node
	comment string
}

func newNode() node {
	return node{}
}

func (n *node) Parent() Node { return n.parent }

func (n *node) SetParent(parent Node) {
	n.parent = parent
}

func (n *node) Comment() string           { return n.comment }
func (n *node) SetComment(comment string) { n.comment = comment }

// status indicate the status of node's runtime.
type status int8

const (
	// The initail state.
	sNone = status(iota)

	// Running.
	sRunning

	// Terminated.
	sTerminated

	// Was stopped.
	sStopped

	// Was destroyed.
	sDestroyed
)

// The strings represent the status values.
var statusStrings = [...]string{
	sNone:       "none",
	sRunning:    "running",
	sTerminated: "terminated",
	sStopped:    "stopped",
	sDestroyed:  "destroyed",
}

func (s status) String() string { return statusStrings[s] }

// lazyStop indicate node's runtime how to stop.
type lazyStop int8

const (
	// Don't need to stop.
	lzsNone = lazyStop(iota)

	// Stop before update.
	lzsBeforeUpdate

	// Stop after update.
	lzsAfterUpdate
)

// The strings represent the lazyStop values.
var lazyStopStrings = [...]string{
	lzsNone:         "none",
	lzsBeforeUpdate: "before-Update",
	lzsAfterUpdate:  "after-Update",
}

func (l lazyStop) String() string { return lazyStopStrings[l] }

// Result represents the running results of node's runtime and even behavior trees.
type Result int8

const (
	// Success can indicate that the behavior ran successfully,
	// or the node made a decision successfully, or the behavior
	// tree ran successfully.
	Success = Result(iota)

	// Failure can indicate that the behavior fails to run, or
	// the node fails to make a decision, or the behavior tree
	// fails to run.
	Failure

	// Running can indicate that a behavior run is running, or
	// that a node is making a decision, or that the behavior
	// tree is running.
	Running
)

// The strings repesents the Result values.
var resultStrings = [...]string{
	Success: "success",
	Failure: "failure",
	Running: "running",
}

func (r Result) String() string { return resultStrings[r] }

// TaskType indicate how the task will run.
type TaskType int8

const (
	// Single task, no any subtask.
	Single = TaskType(iota)

	// Serial task, there are subtasks and the subtasks run one
	// by one.
	Serial

	// Parallel task, there are subtasks and the subtasks run
	// together.
	Parallel
)

var taskTypeStrings = [...]string{
	Single:   "single",
	Serial:   "serial",
	Parallel: "parallel",
}

func (tt TaskType) String() string { return taskTypeStrings[tt] }

// Task represents the independent parts of behavir tree node.
// Task maintains runtime data and implements the logic of the
// corresponding node.
type Task interface {
	// Get the TaskType.
	TaskType() TaskType

	// OnCreate is called immediately after the Task is created.
	// node indicates the node on which the Task is created.
	OnCreate(node Node)

	// OnDestroy is called before the Task is destroyed.
	OnDestroy()

	// OnInit is called before the first update of the Task.
	// childNodes is used to return the child nodes that need
	// to run next. ctx represents the running context of the
	// behavior tree.
	OnInit(childNodes *NodeList, ctx *Context) bool

	// OnUpdate is called until the Task is terminated.
	OnUpdate(ctx *Context) Result

	// OnTerminate is called after ths last update of the Task.
	OnTerminate(ctx *Context)

	// OnChildTerminated is called when a sub Task is terminated.
	//
	// result Indicates the running result of the subtask.
	// childNodes is used to return the child nodes that need to
	// run next.
	//
	// OnChildTerminated returns the decision result.
	OnChildTerminated(result Result, childNodes *NodeList, ctx *Context) Result
}

// agent represents common parts of behavior tree node.
// agent links Node and Task, maintains status infomation
// and implements the workflow of behavior tree node.
// All ruuning agents form a run-time behavior tree.
type agent struct {
	// Corresponding Node.
	node Node

	// Corresponding Task.
	task Task

	// Parent agent.
	parent *agent

	// Child agent list.
	firstChild *agent

	// Previous, next agent.
	prev, next *agent

	// Store the serial number of the latest updating.
	latestUpdateSeri uint32

	// Store the current status.
	st status

	// Store the lazyStop type.
	lzStop lazyStop

	// agent placeholder int the work queue.
	elem *element
}

// onCreate is called immediately after the agent is created.
func (a *agent) onCreate(node Node, task Task) {
	a.node = node
	a.task = task
	a.latestUpdateSeri = 0
	a.st = sNone
	a.lzStop = lzsNone
}

// onDestroy is called before the agent is destroyed.
func (a *agent) onDestroy() {
	a.node = nil
	a.task = nil
}

// Indicates whether the agent is persistent. That is the
// the update method of the agent must be called whenever
// the behavior tree update before it terminated.
func (a *agent) isPersistent() bool { return a.task.TaskType() == Single }

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

// Add a running child for the agent.
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

// Remove a child for the agent.
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

// Running logic of the agent.
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
		if !a.task.OnInit(ctx.getChildNodeList(), ctx) {
			a.task.OnTerminate(ctx)
			a.setStatus(sTerminated)
			return Failure
		}

		if debug {
			switch a.task.TaskType() {
			case Single:
				assert.AssertF(ctx.getChildNodeList().Len() == 0, "node type \"%s\" has children", a.node.NodeType().String())

			case Serial:
				assert.AssertF(ctx.getChildNodeList().Len() == 1, "node type \"%s\" have no or more than one child", a.node.NodeType().String())

			case Parallel:
				assert.AssertF(ctx.getChildNodeList().Len() > 0, "node type \"%s\" have no children", a.node.NodeType().String())
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

	if result == Running {
		a.setStatus(sRunning)
	} else {
		// terminate.
		a.task.OnTerminate(ctx)
		a.setStatus(sTerminated)
	}

	return result
}

// Procoess child nodes filtered by making decision. Child nodes
// are cached in Context.
func (a *agent) processNextChildren(ctx *Context) {
	childNodeList := ctx.getChildNodeList()
	for nextChildNode := childNodeList.pop(); nextChildNode != nil; nextChildNode = childNodeList.pop() {
		childAgent := createAgent(nextChildNode)
		a.addChild(childAgent)
		ctx.pushCurrentAgent(childAgent)
	}
}

// If the agent is running, stop it. remove all child agents,
// notify the task to terminate.
func (a *agent) stop(ctx *Context) {
	if a.getStatus() != sRunning {
		return
	}

	if debug {
		log.Printf("agent nodetype:%v stop", a.node.NodeType())
	}

	child := a.firstChild
	for child != nil {
		agent := child
		child = child.getNext()
		a.removeChild(agent)
	}

	a.task.OnTerminate(ctx)
	a.setStatus(sStopped)
	a.setLZStop(lzsNone)
}

// Lazy-Stop the agent if it is running and not set with
// lazy-stop state yet.
func (a *agent) lazyStop(ctx *Context) {
	if debug {
		log.Printf("agent nodetype:%v lazyStop", a.node.NodeType())
	}

	st := a.getStatus()
	if st == sStopped || st == sTerminated || a.getLZStop() != lzsNone {
		return
	}

	if a.latestUpdateSeri != ctx.getUpdateSeri() {
		// Not updated on the latest updating.
		// Stop after update.
		a.setLZStop(lzsAfterUpdate)
	} else {
		// Updated on the latest updating.
		// Stop before update.
		a.setLZStop(lzsBeforeUpdate)
	}

	// Lazy-Stop need agent to update again.
	if a.elem == nil || a.getLZStop() == lzsBeforeUpdate {
		ctx.pushCurrentAgent(a)
	}
}

// The implementation of Lazy-Stop on agent.
func (a *agent) doLazyStop(ctx *Context) Result {
	a.lazyStopChildren(ctx)
	a.task.OnTerminate(ctx)
	a.setStatus(sStopped)
	a.setLZStop(lzsNone)
	return Failure
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

// onChildTerminated is called when a child agent is terminated.
func (a *agent) onChildTerminated(child *agent, result Result, ctx *Context) Result {
	if debug {
		log.Printf("agent nodetype:%v onChildTerminated %v", a.node.NodeType(), result)
		assert.Assert(a.task.TaskType() != Single, "shouldnt be singletask")
		assert.Assert(child.getParent() == a, "invalid child")
		assert.NotEqual(result, Running, "child terminated with running")
	}

	// Remove child.
	a.removeChild(child)

	// Not running, Failure.
	if a.getStatus() != sRunning {
		return Failure
	}

	// Lazy-Stopping, Running.
	if a.getLZStop() != lzsNone {
		return Running
	}

	// Invoke task.OnChildTerminated to make decision.
	if result = a.task.OnChildTerminated(result, ctx.getChildNodeList(), ctx); result == Running {
		if debug {
			switch a.task.TaskType() {
			case Serial:
				assert.AssertF(ctx.getChildNodeList().Len() == 1, "node type \"%s\" has no or more than one next child", a.node.NodeType().String())

			case Parallel:
				assert.AssertF(ctx.getChildNodeList().Len() == 0, "node type \"%s\" has next children", a.node.NodeType().String())
			}
		}

		a.processNextChildren(ctx)
	} else {
		if debug {
			assert.AssertF(ctx.getChildNodeList().Len() == 0, "node type \"%s\" has next children on terminating.", a.node.NodeType())
		}

		// Lazy-Stop children, avoid nested calls.
		a.lazyStopChildren(ctx)

		a.task.OnTerminate(ctx)
		a.setStatus(sTerminated)
		a.setLZStop(lzsNone)
	}

	return result
}

// The pool to cache destroyed agent.
var agentPool = newPool(func() interface{} { return &agent{} })

// Create agent using node.
func createAgent(node Node) *agent {
	nodeMETA := getNodeMETAByType(node.NodeType())
	if nodeMETA == nil {
		panic(fmt.Sprintf("node type %d meta not found, %s", node.NodeType(), reflect.TypeOf(node).Elem().Name()))
	}

	task := nodeMETA.createTask(node)
	switch task.TaskType() {
	case Single, Serial, Parallel:
	default:
		panic(fmt.Sprintf("node type \"%s\" create invalid type %d task", node.NodeType().String(), task.TaskType()))
	}

	agent := agentPool.get().(*agent)
	agent.onCreate(node, task)

	return agent
}

// Destroy the agent.
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

// Root node, a special node in behavior tree. it has
// only one child and no parent. It returns result of
// child directly.
type rootNode struct {
	child Node
}

func newRootNode() *rootNode {
	return &rootNode{}
}

func (rootNode) NodeType() NodeType { return root }
func (rootNode) Parent() Node       { return nil }
func (rootNode) SetParent(Node)     {}
func (rootNode) Comment() string    { return "" }
func (rootNode) SetComment(string)  {}
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

// rootNode Task.
type rootTask struct {
	node *rootNode
}

// rootNode Task is serail task.
func (r *rootTask) TaskType() TaskType { return Serial }

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

func (r *rootTask) OnUpdate(ctx *Context) Result { return Running }
func (r *rootTask) OnTerminate(ctx *Context)     {}

func (r *rootTask) OnChildTerminated(result Result, nextNodes *NodeList, ctx *Context) Result {
	// Returns result of child directly.
	return result
}

// BevTree, contains the structure data
type BevTree struct {
	// The name of the behavior tree.
	name string

	// The comment of the behavior tree.
	comment string

	// The root node of behavior tree.
	root *rootNode
}

func NewBevTree() *BevTree {
	tree := &BevTree{
		root: newRootNode(),
	}
	return tree
}

func (t *BevTree) Name() string              { return t.name }
func (t *BevTree) SetName(name string)       { t.name = name }
func (t *BevTree) Comment() string           { return t.comment }
func (t *BevTree) SetComment(comment string) { t.comment = comment }

func (t *BevTree) Root() *rootNode { return t.root }

func (t *BevTree) Clear() {
	t.root.SetChild(nil)
}

func (t *BevTree) Update(ctx *Context) Result {
	if ctx.noAgents() {
		// No agents indicate the behavior tree was not run yet
		// or it had completed a traversal from root to root node.
		// Need to start a new traversal from the root node.
		ctx.pushCurrentAgent(createAgent(t.root))
	}

	// Update the Context.
	ctx.update()

	// The default result.
	result := Running

	// Run agent one by one until there are no agents at current
	// updating or back to root node.
	for agent := ctx.popCurrentAgent(); agent != nil; agent = ctx.popCurrentAgent() {
		r := agent.update(ctx)
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

				r = parent.onChildTerminated(agent, r, ctx)
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

			ctx.pushNextAgent(agent)
		}
	}

	assert.Assert(result == Running || ctx.noAgents(), "Update terminated but already has agents")

	return result
}

func (t *BevTree) Stop(ctx *Context) {
	ctx.reset()
}
