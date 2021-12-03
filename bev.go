package bevtree

import (
	"github.com/GodYY/gutils/assert"
)

type BevType int32

type BevParams interface {
	BevType() BevType
}

type Bev interface {
	BevType() BevType
	OnCreate(BevParams)
	OnDestroy()
	OnInit(*Context) bool
	OnUpdate(*Context) Result
	OnTerminate(*Context)
}

type bevMETA struct {
	name          string
	typ           BevType
	bevCreator    func() Bev
	paramsCreator func() BevParams
	bevPool       *pool
}

func (meta *bevMETA) createParams() BevParams {
	return meta.paramsCreator()
}

func (meta *bevMETA) createBev(params BevParams) Bev {
	b := meta.bevPool.get().(Bev)
	if b != nil {
		b.OnCreate(params)
	}
	return b
}

func (meta *bevMETA) destroyBev(b Bev) {
	b.OnDestroy()
	meta.bevPool.put(b)
}

var bevName2META = map[string]*bevMETA{}
var bevType2META = map[BevType]*bevMETA{}

func getBevMETAByType(bevType BevType) *bevMETA { return bevType2META[bevType] }

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

type BevNode struct {
	node
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

func (b *bevTask) OnInit(_ *NodeList, ctx *Context) bool {
	if b.bev == nil {
		return false
	} else {
		return b.bev.OnInit(ctx)
	}
}

func (b *bevTask) OnUpdate(ctx *Context) Result {
	if b.bev == nil {
		return Failure
	} else {
		return b.bev.OnUpdate(ctx)
	}
}

func (b *bevTask) OnTerminate(ctx *Context) {
	if b.bev != nil {
		b.bev.OnTerminate(ctx)
	}
}

func (b *bevTask) OnChildTerminated(Result, *NodeList, *Context) Result { panic("shouldnt be invoked") }
