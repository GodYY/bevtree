package bevtree

import (
	"github.com/GodYY/gutils/assert"
)

// Behavior type.
type BevType int32

// Behavior parameters, the structure data of Behavior.
type BevParams interface {
	BevType() BevType
}

// The interface behavior must implements.
type Bev interface {
	// Behavior type.
	BevType() BevType

	// OnCreate is called immediately after the behavior is created.
	OnCreate(BevParams)

	// OnDestroy is called before the behavior is destroyed.
	OnDestroy()

	// OnInit is called before the first update of the behavior.
	OnInit(Context) bool

	// OnUpdate is called when the behavior tree update before the
	// behavior terminate.
	OnUpdate(Context) Result

	// OnTerminate is called after the last update of the behavior.
	OnTerminate(Context)
}

// The metadata of behavior.
type bevMETA struct {
	// Behavior name.
	name string

	// Behaivor type, assigned at registration.
	typ BevType

	// The creator of the behavior.
	bevCreator func() Bev

	// The paramters creator of behavior.
	paramsCreator func() BevParams

	// The pool that cache destroyed behavior.
	bevPool *pool
}

// Use paramsCreator to create behavior parameters.
func (meta *bevMETA) createParams() BevParams {
	return meta.paramsCreator()
}

// Create a behavior. First, get one cached or create a new one.
// Then, call the OnCreate method of the behavior.
func (meta *bevMETA) createBev(params BevParams) Bev {
	b := meta.bevPool.get().(Bev)
	if b != nil {
		b.OnCreate(params)
	}
	return b
}

// Destroy the behavior. First, call the OnDestroy method of the
// behavior. Then, put it to be cacehd to the pool.
func (meta *bevMETA) destroyBev(b Bev) {
	b.OnDestroy()
	meta.bevPool.put(b)
}

// The mapping of behavior name to metadata.
var bevName2META = map[string]*bevMETA{}

// The mapping of behavior type to metadata.
var bevType2META = map[BevType]*bevMETA{}

// Get behavior metadata by behavior type.
func getBevMETAByType(bevType BevType) *bevMETA { return bevType2META[bevType] }

// Register a type of behavior. It create the metadata of the
// behavior, and assign it a type value.
func RegisterBevType(name string, bevCreator func() Bev, paramsCreator func() BevParams) BevType {
	assert.NotEqual(name, "", "invalid name")
	assert.Assert(bevCreator != nil, "bevCreator nil")
	assert.Assert(paramsCreator != nil, "paramsCreator nil")

	assert.AssertF(bevName2META[name] == nil, "bev type \"%s\" already registered", name)

	meta := &bevMETA{
		name:          name,
		typ:           BevType(len(bevName2META)),
		bevCreator:    bevCreator,
		paramsCreator: paramsCreator,
		bevPool:       newPool(func() interface{} { return bevCreator() }),
	}

	bevName2META[name] = meta
	bevType2META[meta.typ] = meta

	return meta.typ
}

func (t BevType) String() string { return bevType2META[t].name }

func chekcBevTyps() {
	for _, v := range bevName2META {
		params := v.createParams()
		assert.AssertF(params != nil, "bev type \"%s\" create nil BevParams", v.name)
		assert.AssertF(params.BevType() == v.typ, "BevParams created of type \"%s\" has different type", v.name)

		bev := v.createBev(params)
		assert.AssertF(bev != nil, "bev type \"%s\" create nil Bev", v.name)
		assert.AssertF(bev.BevType() == v.typ, "Bev created of type \"%s\" has different type", v.name)
		v.destroyBev(bev)
	}
}

func init() {
	chekcBevTyps()
}

// The behavior node of behavior tree, a kind of leaf node.
type BevNode struct {
	// Common part of node.
	node

	// Behavior parameters.
	bevParams BevParams
}

func NewBevNode(bevParams BevParams) *BevNode {
	return &BevNode{
		node:      newNode(),
		bevParams: bevParams,
	}
}

func (BevNode) NodeType() NodeType { return behavior }

func (b *BevNode) BevParams() BevParams             { return b.bevParams }
func (b *BevNode) SetBevParams(bevParams BevParams) { b.bevParams = bevParams }

// Behavior task, the runtime of BevNode.
type bevTask struct {
	bev Bev
}

func (b *bevTask) TaskType() TaskType { return Single }

func (b *bevTask) OnCreate(node Node) {
	bevNode := node.(*BevNode)
	bevParams := bevNode.BevParams()
	if bevParams != nil {
		b.bev = getBevMETAByType(bevParams.BevType()).createBev(bevParams)
	}
}

func (b *bevTask) OnDestroy() {
	if b.bev != nil {
		getBevMETAByType(b.bev.BevType()).destroyBev(b.bev)
		b.bev = nil
	}
}

func (b *bevTask) OnInit(_ NodeList, ctx Context) bool {
	if b.bev == nil {
		return false
	} else {
		return b.bev.OnInit(ctx)
	}
}

func (b *bevTask) OnUpdate(ctx Context) Result {
	if b.bev == nil {
		return Failure
	} else {
		return b.bev.OnUpdate(ctx)
	}
}

func (b *bevTask) OnTerminate(ctx Context) {
	if b.bev != nil {
		b.bev.OnTerminate(ctx)
	}
}

func (b *bevTask) OnChildTerminated(Result, NodeList, Context) Result { panic("shouldnt be invoked") }
