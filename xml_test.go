package bevtree

import (
	"encoding/xml"
	"math/rand"
	"testing"
	"time"
)

type bevBBIncr struct {
	key     string
	limited int
	count   int
}

func newBevBBIncr(key string, limited int) *bevBBIncr {
	return &bevBBIncr{
		key:     key,
		limited: limited,
	}
}

var btBBIncr = RegisterBevType("blackboardIncr", func() Bev {
	return &bevBBIncr{}
})

func (b *bevBBIncr) BevType() BevType { return btBBIncr }
func (b *bevBBIncr) OnCreate(template Bev) {
	tmpl := template.(*bevBBIncr)
	b.key = tmpl.key
	b.limited = tmpl.limited
	b.count = 0
}
func (b *bevBBIncr) OnDestroy()             {}
func (b *bevBBIncr) OnInit(_ *Context) bool { return true }

func (b *bevBBIncr) OnUpdate(e *Context) Result {
	e.IncInt(b.key)
	b.count++

	if b.count >= b.limited {
		return RSuccess
	} else {
		return RRunning
	}

}

func (b *bevBBIncr) OnTerminate(_ *Context) {}

func (b *bevBBIncr) Clone() Bev {
	copy := *b
	return &copy
}

func (b *bevBBIncr) Destroy() {}

func (b *bevBBIncr) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	var err error

	if err = e.EncodeElement(b.key, xml.StartElement{Name: createXMLName("key")}); err != nil {
		return err
	}

	return e.EncodeElement(b.limited, xml.StartElement{Name: createXMLName("limited")})
}

func (b *bevBBIncr) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	return d.DecodeEndTo(start.End(), func(d *XMLDecoder, start xml.StartElement) error {
		if start.Name == createXMLName("key") {
			return d.DecodeElement(&b.key, start)
		}

		if start.Name == createXMLName("limited") {
			return d.DecodeElement(&b.limited, start)
		}

		return nil
	})
}

var xmlNameKey = createXMLName("key")

func TestBevTreeMarshalXML(t *testing.T) {
	key := "key"
	sum := 0

	unit := 1

	rand.Seed(time.Now().UnixNano())

	tree := NewBevTree()
	paral := NewParallelNode()
	tree.Root().SetChild(paral)

	bd := newBevBBIncr(key, unit)

	sc := NewSucceederNode()
	sc.SetChild(NewBevNode(bd))
	paral.AddChild(sc)
	sum += unit

	low := 5
	max := 10
	rtimes := low + rand.Intn(max-low+1)
	r := NewRepeaterNode(rtimes)
	r.SetChild(NewBevNode(bd))
	paral.AddChild(r)
	sum += rtimes * unit

	iv_sc := NewSucceederNode()
	iv := NewInverterNode()
	iv.SetChild(NewBevNode(bd))
	iv_sc.SetChild(iv)
	paral.AddChild(iv_sc)
	sum += unit

	ruf := NewRepeatUntilFailNode(true)
	ruf_iv := NewInverterNode()
	ruf.SetChild(ruf_iv)
	ruf_iv.SetChild(NewBevNode(bd))
	paral.AddChild(ruf)
	sum += unit

	seqTimes := low + rand.Intn(max-low+1)
	seq := NewSequenceNode()
	for i := 0; i < seqTimes; i++ {
		seq.AddChild(NewBevNode(bd))
	}
	paral.AddChild(seq)
	sum += seqTimes * unit

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
	sum += (selcSuccN + 1) * unit

	// paral.AddChild(NewRandSequence())
	// paral.AddChild(NewRandSelector())
	// paral.AddChild(NewParallel())

	ctx := NewContext(nil)
	ctx.Set(key, 0)
	tree.Update(ctx)
	v, _ := ctx.GetInt(key)
	if v != sum {
		t.Fatalf("test BevTree before marshal: sum(%d) != %d", v, sum)
	}

	data, err := MarshalXMLBevTree(tree)
	if err != nil {
		t.Fatal("marshal BevTree:", err)
	} else {
		t.Log("marshal BevTree:", string(data))
	}

	newTree := new(BevTree)
	if err := UnmarshalXMLBevTree(data, newTree); err != nil {
		t.Fatal("unmarshal previos BevTree:", err)
	}

	ctx.Set(key, 0)
	newTree.Update(ctx)

	v, _ = ctx.GetInt(key)
	if v != sum {
		t.Fatalf("test BevTree after unmarshal: sum(%d) != %d", v, sum)
	}

}
