package bevtree

import (
	"encoding/xml"

	"github.com/GodYY/gutils/assert"
)

type BevType int32

type Bev interface {
	BevType() BevType
	OnCreate(template Bev)
	OnDestroy()
	OnInit(*Context) bool
	OnUpdate(*Context) Result
	OnTerminate(*Context)
	MarshalBTXML(*XMLEncoder, xml.StartElement) error
	UnmarshalBTXML(*XMLDecoder, xml.StartElement) error
}

type bevMETA struct {
	name    string
	typ     BevType
	creator func() Bev
	bevPool *pool
}

func (meta *bevMETA) createTemplate() Bev {
	return meta.creator()
}

func (meta *bevMETA) createBev(template Bev) Bev {
	b := meta.bevPool.get().(Bev)
	b.OnCreate(template)
	return b
}

func (meta *bevMETA) destroyBev(b Bev) {
	b.OnDestroy()
	meta.bevPool.put(b)
}

var bevName2META = map[string]*bevMETA{}
var bevType2META = map[BevType]*bevMETA{}

func getBevMETAByType(bevType BevType) *bevMETA { return bevType2META[bevType] }

func RegisterBevType(name string, creator func() Bev) BevType {
	assert.NotEqual(name, "", "invalid name")
	assert.Assert(creator != nil, "creator nil")

	assert.AssertF(bevName2META[name] == nil, "bev type \"%s\" already registered", name)

	meta := &bevMETA{
		name:    name,
		typ:     BevType(len(bevName2META)),
		creator: creator,
		bevPool: newPool(func() interface{} { return creator() }),
	}

	bevName2META[name] = meta
	bevType2META[meta.typ] = meta

	return meta.typ
}

func (t BevType) String() string { return bevType2META[t].name }

func chekcBevTyps() {
	for _, v := range bevName2META {
		bev := v.createTemplate()
		assert.AssertF(bev != nil, "bev type \"%s\" create nil bev", v.name)
		assert.AssertF(bev.BevType() == v.typ, "bev created of type \"%s\" has different type", v.name)
	}
}

func init() {
	chekcBevTyps()
}

type BevNode struct {
	node
	bev Bev
}

func newBevNode() *BevNode {
	return &BevNode{
		node: newNode(),
	}
}

func NewBevNode(bev Bev) *BevNode {
	assert.Assert(bev != nil, "bev nil")

	b := newBevNode()
	b.bev = bev
	return b
}

func (BevNode) NodeType() NodeType { return behavior }

type bevTask struct {
	bev Bev
}

func (b *bevTask) TaskType() TaskType { return TaskSingle }

func (b *bevTask) OnCreate(node Node) {
	bevNode := node.(*BevNode)
	if bevNode.bev != nil {
		b.bev = getBevMETAByType(bevNode.bev.BevType()).createBev(bevNode.bev)
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
		return RFailure
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
