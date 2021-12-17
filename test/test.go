package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	. "github.com/GodYY/bevtree"
)

const (
	function       = BevType("func")
	increase       = BevType("incr")
	update         = BevType("update")
	blackboardIncr = BevType("blackboardIncr")
)

type bevFunc struct {
	f func(Context) Result
}

func newBevFunc(f func(Context) Result) *bevFunc {
	return &bevFunc{f: f}
}

func (bevFunc) BevType() BevType { return function }

func (b *bevFunc) CreateInstance() BevInstance {
	return &bevFuncEntity{f: b.f}
}

func (b *bevFunc) DestroyInstance(BevInstance) {}

type bevFuncEntity struct {
	f func(Context) Result
}

func (b *bevFuncEntity) BevType() BevType { return function }

func (b *bevFuncEntity) OnInit(e Context) bool { return true }

func (b *bevFuncEntity) OnUpdate(e Context) Result {
	// fmt.Println("behaviorFunc OnUpdate")
	return b.f(e)
}

func (b *bevFuncEntity) OnTerminate(e Context) {
}

type behaviorIncr struct {
	key     string
	limited int
}

func newBehaviorIncr(key string, limited int) *behaviorIncr {
	return &behaviorIncr{key: key, limited: limited}
}

func (behaviorIncr) BevType() BevType { return increase }

func (b *behaviorIncr) CreateInstance() BevInstance {
	return &behaviorIncrEntity{behaviorIncr: b}
}

func (b *behaviorIncr) DestroyInstance(BevInstance) {}

type behaviorIncrEntity struct {
	*behaviorIncr
	count int
}

func (b *behaviorIncrEntity) BevType() BevType { return increase }

func (b *behaviorIncrEntity) OnInit(e Context) bool { return true }

func (b *behaviorIncrEntity) OnUpdate(e Context) Result {
	if b.count >= b.limited {
		return Failure
	}

	b.count++
	e.DataSet().IncInt(b.key)
	if b.count >= b.limited {
		return Success
	}

	return Running
}

func (b *behaviorIncrEntity) OnTerminate(e Context) { b.count = 0 }

type behaviorUpdate struct {
	limited int
}

func newBehaviorUpdate(lmited int) *behaviorUpdate {
	return &behaviorUpdate{limited: lmited}
}

func (behaviorUpdate) BevType() BevType { return update }

func (b *behaviorUpdate) CreateInstance() BevInstance {
	return &behaviorUpdateEntity{behaviorUpdate: b}
}

func (b *behaviorUpdate) DestroyInstance(BevInstance) {}

type behaviorUpdateEntity struct {
	*behaviorUpdate
	count int
}

func (b *behaviorUpdateEntity) BevType() BevType { return update }

func (b *behaviorUpdateEntity) OnInit(e Context) bool { return true }

func (b *behaviorUpdateEntity) OnUpdate(e Context) Result {
	if b.count >= b.limited {
		return Success
	}

	b.count++
	if b.count >= b.limited {
		return Success
	}

	return Running
}

func (b *behaviorUpdateEntity) OnTerminate(e Context) { b.count = 0 }

func init() {

}

type bevBBIncr struct {
	Key     string
	Limited int
}

func newBevBBIncr(key string, limited int) *bevBBIncr {
	return &bevBBIncr{
		Key:     key,
		Limited: limited,
	}
}

func (bevBBIncr) BevType() BevType { return blackboardIncr }

func (b *bevBBIncr) CreateInstance() BevInstance {
	return &bevBBIncrEntity{bevBBIncr: b}
}

func (b *bevBBIncr) DestroyInstance(BevInstance) {}

type bevBBIncrEntity struct {
	*bevBBIncr
	count int
}

func (b *bevBBIncrEntity) BevType() BevType      { return blackboardIncr }
func (b *bevBBIncrEntity) OnInit(_ Context) bool { return true }

func (b *bevBBIncrEntity) OnUpdate(e Context) Result {
	e.DataSet().IncInt(b.Key)
	b.count++

	if b.count >= b.Limited {
		return Success
	} else {
		return Running
	}

}

func (b *bevBBIncrEntity) OnTerminate(_ Context) {}

func newTestFramework() *Framework {
	framework := NewFramework()
	framework.RegisterBevType(function, func() Bev { return new(bevFunc) })
	framework.RegisterBevType(increase, func() Bev { return new(behaviorIncr) })
	framework.RegisterBevType(update, func() Bev { return new(behaviorUpdate) })
	framework.RegisterBevType(blackboardIncr, func() Bev { return &bevBBIncr{} })
	return framework
}

func main() {
	framework := newTestFramework()
	configPath := "./config.xml"
	exporter := NewExporter(framework)
	exporter.SetLoadAll(true)

	tree := NewTree("test subtree")

	// parallel := NewParallelNode()
	// tree.Root().SetChild(parallel)

	weightSelector := NewWeightSelectorNode()
	tree.Root().SetChild(weightSelector)

	key := "key"
	// sum := 0
	unit := 1
	low := 5
	max := 10

	rand.Seed(time.Now().UnixNano())

	sum_a := 0
	{
		subtree_a := NewTree("subtree_a")
		exporter.AddTree(subtree_a, "subtree_a.xml")

		weightSelector.AddChild(NewSubtreeNode(subtree_a, false), 0.199)
		paral := NewParallelNode()
		subtree_a.Root().SetChild(paral)

		bd := newBevBBIncr(key, unit)

		sc := NewSucceederNode()
		sc.SetChild(NewBevNode(bd))
		paral.AddChild(sc)
		sum_a += unit

		rtimes := low + rand.Intn(max-low+1)
		r := NewRepeaterNode(rtimes)
		r.SetChild(NewBevNode(bd))
		paral.AddChild(r)
		sum_a += rtimes * unit

		iv_sc := NewSucceederNode()
		iv := NewInverterNode()
		iv.SetChild(NewBevNode(bd))
		iv_sc.SetChild(iv)
		paral.AddChild(iv_sc)
		sum_a += unit

		ruf := NewRepeatUntilFailNode(true)
		ruf_iv := NewInverterNode()
		ruf.SetChild(ruf_iv)
		ruf_iv.SetChild(NewBevNode(bd))
		paral.AddChild(ruf)
		sum_a += unit

		seqTimes := low + rand.Intn(max-low+1)
		seq := NewSequenceNode()
		for i := 0; i < seqTimes; i++ {
			seq.AddChild(NewBevNode(bd))
		}
		paral.AddChild(seq)
		sum_a += seqTimes * unit

		selcTimes := low + rand.Intn(max-low+1)
		selc := NewSelectorNode()
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
		sum_a += (selcSuccN + 1) * unit
	}

	sum_b := 0
	{
		subtree_b := NewTree("subtree_b")
		exporter.AddTree(subtree_b, "subtree_b.xml")

		weightSelector.AddChild(NewSubtreeNode(subtree_b, false), 0.801)
		paral := NewParallelNode()
		subtree_b.Root().SetChild(paral)

		bd := newBevBBIncr(key, unit)

		sc := NewSucceederNode()
		sc.SetChild(NewBevNode(bd))
		paral.AddChild(sc)
		sum_b += unit

		rtimes := low + rand.Intn(max-low+1)
		r := NewRepeaterNode(rtimes)
		r.SetChild(NewBevNode(bd))
		paral.AddChild(r)
		sum_b += rtimes * unit

		iv_sc := NewSucceederNode()
		iv := NewInverterNode()
		iv.SetChild(NewBevNode(bd))
		iv_sc.SetChild(iv)
		paral.AddChild(iv_sc)
		sum_b += unit

		ruf := NewRepeatUntilFailNode(true)
		ruf_iv := NewInverterNode()
		ruf.SetChild(ruf_iv)
		ruf_iv.SetChild(NewBevNode(bd))
		paral.AddChild(ruf)
		sum_b += unit

		seqTimes := low + rand.Intn(max-low+1)
		seq := NewSequenceNode()
		for i := 0; i < seqTimes; i++ {
			seq.AddChild(NewBevNode(bd))
		}
		paral.AddChild(seq)
		sum_b += seqTimes * unit

		selcTimes := low + rand.Intn(max-low+1)
		selc := NewSelectorNode()
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
		sum_b += (selcSuccN + 1) * unit
	}

	exporter.AddTree(tree, "test_subtree.xml")

	if err := exporter.Export(configPath); err != nil {
		log.Fatal(err)
	}

	if err := framework.Init(configPath); err != nil {
		log.Fatal(err)
	}

	entity, err := framework.CreateEntity("test subtree", nil)
	if err != nil {
		log.Fatal(err)
	}

	entity.Context().DataSet().Set(key, 0)
	entity.Update()
	if val, ok := entity.Context().DataSet().GetInt(key); !ok {
		log.Fatal("value not exist")
	} else if val != sum_a && val != sum_b && val != 0 {
		log.Fatalf("val(%d) != [%d, %d, 0]", val, sum_a, sum_b)
	} else {
		fmt.Println(val, sum_a, sum_b)
		fmt.Println("success")
	}

}
