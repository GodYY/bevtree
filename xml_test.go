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

func (b *bevBBIncr) OnInit(_ *Env) {
}

func (b *bevBBIncr) OnUpdate(e *Env) Result {
	val := e.Val(b.key).(int)
	val++
	e.Set(b.key, val)

	b.count++

	if b.count >= b.limited {
		return RSuccess
	} else {
		return RRunning
	}

}

func (b *bevBBIncr) OnTerminate(_ *Env) {
}

type bevBBIncrDef struct {
	key     string
	limited int
}

func newBevBBIncrDef(key string, limited int) *bevBBIncrDef {
	return &bevBBIncrDef{key: key, limited: limited}
}

func (bd *bevBBIncrDef) CreateBev() BevInst {
	return newBevBBIncr(bd.key, bd.limited)
}

func (bd *bevBBIncrDef) DestroyBev(_ BevInst) {
}

var xmlNameKey = createXMLName("key")

func (bd *bevBBIncrDef) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error {
	var err error

	if err = e.EncodeElement(bd.key, xml.StartElement{Name: xmlNameKey}); err != nil {
		return err
	}

	return e.EncodeElement(bd.limited, xml.StartElement{Name: xmlNameLimited})
}

func (bd *bevBBIncrDef) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error {
	return d.DecodeEndTo(start.End(), func(d *XMLDecoder, start xml.StartElement) error {
		if start.Name == xmlNameKey {
			return d.DecodeElement(&bd.key, start)
		}

		if start.Name == xmlNameLimited {
			return d.DecodeElement(&bd.limited, start)
		}

		return nil
	})
}

type bevCoder struct {
}

func (c *bevCoder) EncodeXMLBev(e *XMLEncoder, bd Bev, start xml.StartElement) error {
	return e.EncodeStartEnd(bd, start)
}

func (c *bevCoder) DecodeXMLBev(d *XMLDecoder, pbd *Bev, start xml.StartElement) error {
	bd := &bevBBIncrDef{}
	if err := d.DecodeElement(bd, start); err != nil {
		return err
	}

	*pbd = bd
	return nil
}

func TestBevTreeMarshalXML(t *testing.T) {
	key := "key"
	sum := 0
	low := 5
	max := 10
	unit := 1

	rand.Seed(time.Now().UnixNano())

	tree := NewBevTree()
	paral := NewParallel()
	tree.root().SetChild(paral)

	bd := newBevBBIncrDef(key, unit)

	sc := NewSucceeder()
	sc.SetChild(NewBev(bd))
	paral.AddChild(sc)
	sum += unit

	rtimes := low + rand.Intn(max-low+1)
	r := NewRepeater(rtimes)
	r.AddChild(NewBev(bd))
	paral.AddChild(r)
	sum += rtimes * unit

	iv_sc := NewSucceeder()
	iv := NewInverter()
	iv.SetChild(NewBev(bd))
	iv_sc.SetChild(iv)
	paral.AddChild(iv_sc)
	sum += unit

	ruf := NewRepeatUntilFail(true)
	ruf_iv := NewInverter()
	ruf.SetChild(ruf_iv)
	ruf_iv.SetChild(NewBev(bd))
	paral.AddChild(ruf)
	sum += unit

	seqTimes := low + rand.Intn(max-low+1)
	seq := NewSequence()
	for i := 0; i < seqTimes; i++ {
		seq.AddChild(NewBev(bd))
	}
	paral.AddChild(seq)
	sum += seqTimes * unit

	selcTimes := low + rand.Intn(max-low+1)
	selc := NewSelector()
	selcSuccN := rand.Intn(selcTimes)
	for i := 0; i < selcTimes; i++ {
		if selcSuccN == i {
			selc.AddChild(NewBev(bd))
		} else {
			iv := NewInverter()
			iv.AddChild(NewBev(bd))
			selc.AddChild(iv)
		}
	}
	paral.AddChild(selc)
	sum += (selcSuccN + 1) * unit

	// paral.AddChild(NewRandSequence())
	// paral.AddChild(NewRandSelector())
	// paral.AddChild(NewParallel())

	env := NewEnv(nil)
	env.Set(key, 0)
	tree.Update(env)
	if env.Val(key).(int) != sum {
		t.Fatalf("test BevTree before marshal: sum(%d) != %d", env.Val(key).(int), sum)
	}

	data, err := MarshalXMLBevTree(tree, &bevCoder{})
	if err != nil {
		t.Fatal("marshal BevTree:", err)
	} else {
		t.Log("marshal BevTree:", string(data))
	}

	newTree := new(BevTree)
	if err := UnmarshalXMLBevTree(data, newTree, &bevCoder{}); err != nil {
		t.Fatal("unmarshal previos BevTree:", err)
	}

	env.Set(key, 0)
	newTree.Update(env)

	if env.Val(key).(int) != sum {
		t.Fatalf("test BevTree after unmarshal: sum(%d) != %d", env.Val(key).(int), sum)
	}

}
