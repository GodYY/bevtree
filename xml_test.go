package bevtree

import (
	"math/rand"
	"testing"
	"time"
)

type bevBBInccParams struct {
	Key     string
	Limited int
}

func newBevBBIncrParams(key string, limited int) *bevBBInccParams {
	return &bevBBInccParams{
		Key:     key,
		Limited: limited,
	}
}

func (bevBBInccParams) BevType() BevType { return btBBIncr }

type bevBBIncr struct {
	*bevBBInccParams
	count int
}

var btBBIncr = RegisterBevType("blackboardIncr",
	func() Bev {
		return &bevBBIncr{}
	},
	func() BevParams {
		return &bevBBInccParams{}
	},
)

func (b *bevBBIncr) BevType() BevType { return btBBIncr }
func (b *bevBBIncr) OnCreate(desc BevParams) {
	b.bevBBInccParams = desc.(*bevBBInccParams)
	b.count = 0
}
func (b *bevBBIncr) OnDestroy()            {}
func (b *bevBBIncr) OnInit(_ Context) bool { return true }

func (b *bevBBIncr) OnUpdate(e Context) Result {
	e.DataSet().IncInt(b.Key)
	b.count++

	if b.count >= b.Limited {
		return Success
	} else {
		return Running
	}

}

func (b *bevBBIncr) OnTerminate(_ Context) {}

var xmlNameKey = XMLName("key")

func TestBevTreeMarshalXML(t *testing.T) {
	key := "key"
	sum := 0

	unit := 1

	rand.Seed(time.Now().UnixNano())

	tree := NewTree()
	tree.SetName("XML测试")
	tree.SetComment("用于XML测试的行为树")
	paral := NewParallelNode()
	paral.SetComment("并行测试")
	tree.Root().SetChild(paral)

	bd := newBevBBIncrParams(key, unit)

	sc := NewSucceederNode()
	sc.SetComment("succeeder测试")
	sc.SetChild(NewBevNode(bd))
	paral.AddChild(sc)
	sum += unit

	low := 5
	max := 10
	rtimes := low + rand.Intn(max-low+1)
	r := NewRepeaterNode(rtimes)
	r.SetComment("repeater测试")
	r.SetChild(NewBevNode(bd))
	paral.AddChild(r)
	sum += rtimes * unit

	iv_sc := NewSucceederNode()
	iv_sc.SetComment("succeeder+inverter测试")
	iv := NewInverterNode()
	iv.SetChild(NewBevNode(bd))
	iv_sc.SetChild(iv)
	paral.AddChild(iv_sc)
	sum += unit

	ruf := NewRepeatUntilFailNode(true)
	ruf.SetComment("repeatuntilfail+inverter测试")
	ruf_iv := NewInverterNode()
	ruf.SetChild(ruf_iv)
	ruf_iv.SetChild(NewBevNode(bd))
	paral.AddChild(ruf)
	sum += unit

	seqTimes := low + rand.Intn(max-low+1)
	seq := NewSequenceNode()
	seq.SetComment("sequence测试")
	for i := 0; i < seqTimes; i++ {
		seq.AddChild(NewBevNode(bd))
	}
	paral.AddChild(seq)
	sum += seqTimes * unit

	selcTimes := low + rand.Intn(max-low+1)
	selc := NewSelectorNode()
	selc.SetComment("selector测试")
	selcSuccN := rand.Intn(selcTimes)
	for i := 0; i < selcTimes; i++ {
		if selcSuccN == i {
			selc.AddChild(NewBevNode(bd))
		} else {
			iv := NewInverterNode()
			iv.SetChild(NewBevNode(bd))
			selc.AddChild(iv)
		}
	}
	paral.AddChild(selc)
	sum += (selcSuccN + 1) * unit

	// paral.AddChild(NewRandSequence())
	// paral.AddChild(NewRandSelector())
	// paral.AddChild(NewParallel())

	entity := NewEntity(tree, nil)
	entity.Context().DataSet().Set(key, 0)
	entity.Update()
	v, _ := entity.Context().DataSet().GetInt(key)
	if v != sum {
		t.Fatalf("test Tree before marshal: sum(%d) != %d", v, sum)
	}
	entity.Release()

	data, err := MarshalXMLBevTree(tree)
	if err != nil {
		t.Fatal("marshal Tree:", err)
	} else {
		t.Log("marshal Tree:", string(data))
	}

	newTree := new(Tree)
	if err := UnmarshalXMLBevTree(data, newTree); err != nil {
		t.Fatal("unmarshal previos Tree:", err)
	}

	entity = NewEntity(newTree, nil)
	entity.Context().DataSet().Set(key, 0)
	entity.Update()

	v, _ = entity.Context().DataSet().GetInt(key)
	if v != sum {
		t.Fatalf("test Tree after unmarshal: sum(%d) != %d", v, sum)
	}

}
