package bevtree

import (
	"encoding/xml"
	"math/rand"
	"testing"
	"time"
)

var (
	btFunc   = RegisterBevType("func", func() Bev { return new(bevFunc) })
	btIncr   = RegisterBevType("incr", func() Bev { return new(behaviorIncr) })
	btUpdate = RegisterBevType("update", func() Bev { return new(behaviorUpdate) })
)

type bevFunc struct {
	f func(*Context) Result
}

func newBevFunc(f func(*Context) Result) *bevFunc {
	return &bevFunc{f: f}
}

func (b *bevFunc) BevType() BevType { return btFunc }

func (b *bevFunc) OnCreate(template Bev) {
	b.f = template.(*bevFunc).f
}

func (b *bevFunc) OnDestroy() {}

func (b *bevFunc) OnInit(e *Context) bool { return true }

func (b *bevFunc) OnUpdate(e *Context) Result {
	// fmt.Println("behaviorFunc OnUpdate")
	return b.f(e)
}

func (b *bevFunc) OnTerminate(e *Context) {
}

func (b *bevFunc) Clone() Bev {
	return newBevFunc(b.f)
}

func (b *bevFunc) Destroy() {}

func (b *bevFunc) MarshalBTXML(e *XMLEncoder, start xml.StartElement) error { return nil }

func (b *bevFunc) UnmarshalBTXML(d *XMLDecoder, start xml.StartElement) error { return nil }

type behaviorIncr struct {
	key     string
	limited int
	count   int
}

func newBehaviorIncr(key string, limited int) *behaviorIncr {
	return &behaviorIncr{
		key:     key,
		limited: limited,
	}
}

func (b *behaviorIncr) BevType() BevType { return btIncr }
func (b *behaviorIncr) OnCreate(template Bev) {
	tmpl := template.(*behaviorIncr)
	b.key = tmpl.key
	b.limited = tmpl.limited
	b.count = 0
}
func (b *behaviorIncr) OnDestroy() {}

func (b *behaviorIncr) OnInit(e *Context) bool { return true }

func (b *behaviorIncr) OnUpdate(e *Context) Result {
	if b.count >= b.limited {
		return RFailure
	}

	b.count++
	e.IncInt(b.key)
	if b.count >= b.limited {
		return RSuccess
	}

	return RRunning
}

func (b *behaviorIncr) OnTerminate(e *Context)                             { b.count = 0 }
func (b *behaviorIncr) MarshalBTXML(*XMLEncoder, xml.StartElement) error   { return nil }
func (b *behaviorIncr) UnmarshalBTXML(*XMLDecoder, xml.StartElement) error { return nil }

type behaviorUpdate struct {
	limited int
	count   int
}

func newBehaviorUpdate(limited int) *behaviorUpdate {
	return &behaviorUpdate{
		limited: limited,
	}
}

func (b *behaviorUpdate) BevType() BevType { return btUpdate }

func (b *behaviorUpdate) OnCreate(template Bev) {
	b.limited = template.(*behaviorUpdate).limited
	b.count = 0
}

func (b *behaviorUpdate) OnDestroy() {}

func (b *behaviorUpdate) OnInit(e *Context) bool { return true }

func (b *behaviorUpdate) OnUpdate(e *Context) Result {
	if b.count >= b.limited {
		return RSuccess
	}

	b.count++
	if b.count >= b.limited {
		return RSuccess
	}

	return RRunning
}

func (b *behaviorUpdate) OnTerminate(e *Context)                             { b.count = 0 }
func (b *behaviorUpdate) MarshalBTXML(*XMLEncoder, xml.StartElement) error   { return nil }
func (b *behaviorUpdate) UnmarshalBTXML(*XMLDecoder, xml.StartElement) error { return nil }

type test struct {
	tree *BevTree
	e    *Context
}

func newTest() *test {
	t := &test{
		tree: NewBevTree(),
		e:    NewContext(nil),
	}
	t.tree = NewBevTree()
	return t
}

func (t *test) run(tt *testing.T, expectedResult Result, expectedKeyValues map[string]interface{}, tick int) {
	result := RRunning

	for i := 0; i < tick; i++ {
		tt.Log("run", i, "start")
		result = RRunning
		k := 0
		for result == RRunning {
			tt.Log("run", i, "update", k)
			k++
			result = t.tree.Update(t.e)
			time.Sleep(1 * time.Millisecond)
		}
		tt.Log("run", i, "end", result)
	}

	if result != expectedResult {
		tt.Fatalf("should return %v but get %v", expectedResult, result)
	}

	for k, v := range expectedKeyValues {
		if t.e.Get(k) != v {
			tt.Fatalf("%s = %v(%v)", k, t.e.Get(k), v)
		}
	}
}

func (t *test) clear() {
	t.tree = nil
	t.e = nil
}

func (t *test) close() {
	t.tree = nil
	t.e.Release()
}

func TestRoot(t *testing.T) {
	test := newTest()
	test.run(t, RFailure, nil, 1)
}

func TestSequence(t *testing.T) {
	test := newTest()

	seq := NewSequenceNode()

	test.tree.Root().SetChild(seq)

	key := "counter"
	test.e.SetInt(key, 0)
	n := 2
	for i := 0; i < n; i++ {
		seq.AddChild(NewBevNode(newBevFunc(func(e *Context) Result {
			e.IncInt(key)
			return RSuccess
		})))
	}

	test.run(t, RSuccess, map[string]interface{}{key: n}, 1)
}

func TestSelector(t *testing.T) {
	test := newTest()

	selc := NewSelectorNode()

	test.tree.Root().SetChild(selc)

	key := "selected"
	var selected int
	n := 10
	for i := 0; i < n; i++ {
		k := i
		selc.AddChild(NewBevNode(newBevFunc((func(e *Context) Result {
			if k == selected {
				test.e.SetInt(key, selected)
				return RSuccess
			} else {
				return RFailure
			}
		}))))
	}

	rand.Seed(time.Now().Unix())
	selected = rand.Intn(n)

	test.run(t, RSuccess, map[string]interface{}{key: selected}, 1)
}

func TestRandomSequence(t *testing.T) {
	rand.Seed(time.Now().Unix())

	test := newTest()

	seq := NewRandSequenceNode()

	test.tree.Root().SetChild(seq)

	key := "counter"
	test.e.SetInt(key, 0)
	n := 2
	for i := 0; i < n; i++ {
		k := i
		seq.AddChild(NewBevNode(newBevFunc(func(e *Context) Result {
			t.Log("seq", k, "update")
			e.IncInt(key)
			return RSuccess
		})))
	}

	test.run(t, RSuccess, map[string]interface{}{key: n}, 1)
}

func TestRandomSelector(t *testing.T) {
	rand.Seed(time.Now().Unix())

	test := newTest()

	selc := NewRandSelectorNode()

	test.tree.Root().SetChild(selc)

	key := "selected"
	var selected int
	n := 10
	for i := 0; i < n; i++ {
		k := i
		selc.AddChild(NewBevNode(newBevFunc((func(e *Context) Result {
			t.Log("seq", k, "update")
			if k == selected {
				test.e.SetInt(key, selected)
				return RSuccess
			} else {
				return RFailure
			}
		}))))
	}

	rand.Seed(time.Now().Unix())
	selected = rand.Intn(n)

	test.run(t, RSuccess, map[string]interface{}{key: selected}, 1)
}

func TestParallel(t *testing.T) {
	test := newTest()

	paral := NewParallelNode()

	test.tree.Root().SetChild(paral)

	rand.Seed(time.Now().Unix())

	n := 2
	for i := 0; i < n; i++ {
		k := i + 1
		timer := time.NewTimer(1000 * time.Millisecond * time.Duration(k))
		paral.AddChild(NewBevNode(newBevFunc(func(e *Context) Result {
			select {
			case <-timer.C:
				t.Logf("timer No.%d up", k)
				return RSuccess
			default:
				t.Logf("timer No.%d update", k)
				return RRunning
			}
		})))
	}

	test.run(t, RSuccess, nil, 1)
}

func TestParallelLazyStop(t *testing.T) {
	test := newTest()

	paral := NewParallelNode()

	test.tree.Root().SetChild(paral)

	rand.Seed(time.Now().Unix())

	lowUpdate, maxUpdate := 2, 10
	n := 10
	lowDepth, maxDepth := 5, 10
	for i := 0; i < n; i++ {
		k := i + 1
		ut := lowUpdate + rand.Intn(maxUpdate-lowUpdate+1)
		c := DecoratorNode(NewInverterNode())
		paral.AddChild(c)

		depth := 5 + rand.Intn(maxDepth-lowDepth)
		for d := 0; d < depth; d++ {
			cc := NewSucceederNode()
			c.SetChild(cc)
			c = cc
		}

		c.SetChild(NewBevNode(newBevFunc(func(e *Context) Result {
			t.Logf("No.%d update", k)
			ut--
			if ut <= 0 {
				t.Logf("No.%d over", k)
				return RSuccess
			} else {
				return RRunning
			}
		})))
	}

	test.run(t, RFailure, nil, 1)
}

func TestRepeater(t *testing.T) {
	test := newTest()

	n := 10
	repeater := NewRepeaterNode(n)

	test.tree.Root().SetChild(repeater)

	key := "counter"
	test.e.SetInt(key, 0)

	repeater.SetChild(NewBevNode(newBevFunc((func(e *Context) Result {
		e.IncInt(key)
		return RSuccess
	}))))

	test.run(t, RSuccess, map[string]interface{}{key: n}, 1)
}

func TestInverter(t *testing.T) {
	test := newTest()

	inverter := NewInverterNode()

	test.tree.Root().SetChild(inverter)

	inverter.SetChild(NewBevNode(newBevFunc(func(e *Context) Result {
		return RFailure
	})))

	test.run(t, RSuccess, nil, 1)
}

func TestSucceeder(t *testing.T) {
	test := newTest()

	succeeder := NewSucceederNode()
	test.tree.Root().SetChild(succeeder)

	succeeder.SetChild(NewBevNode(newBevFunc(func(e *Context) Result { return RFailure })))

	test.run(t, RSuccess, nil, 1)
}

func TestRepeatUntilFail(t *testing.T) {
	test := newTest()

	repeat := NewRepeatUntilFailNode(false)
	test.tree.Root().SetChild(repeat)

	n := 4
	repeat.SetChild(NewBevNode(newBevFunc(func(e *Context) Result {
		t.Log("decr 1")

		n--

		if n <= 0 {
			return RFailure
		}

		return RSuccess
	})))

	test.run(t, RFailure, nil, 1)
}

func TestShareTree(t *testing.T) {

	tree := NewBevTree()
	paral := NewParallelNode()
	tree.Root().SetChild(paral)

	expectedResult := RSuccess
	singleSum := 0
	key := "sum"
	numEnvs := 100
	low, max := 5, 50
	n := 100

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < n; i++ {
		limited := low + rand.Intn(max-low+1)
		singleSum += limited
		t.Logf("singleSum add %d to %d", limited, singleSum)

		paral.AddChild(NewBevNode(newBehaviorIncr(key, limited)))
	}

	envs := make([]*Context, numEnvs)
	for i := 0; i < numEnvs; i++ {
		envs[i] = NewContext(nil)
		envs[i].SetInt(key, 0)
	}

	result := RRunning
	for result == RRunning {
		for i := 0; i < numEnvs; i++ {
			if i > 0 {
				r := tree.Update(envs[i])
				if r != result {
					t.Fatal("invalid result", result, r)
				}
			} else {
				result = tree.Update(envs[i])
			}
		}

		time.Sleep(1 * time.Millisecond)
	}

	if result != expectedResult {
		t.Fatalf("expected %v get %v", expectedResult, result)
	}

	sum := 0
	for i := 0; i < numEnvs; i++ {
		v, _ := envs[i].GetInt(key)
		sum += v
	}

	if sum != singleSum*numEnvs {
		t.Fatalf("expected sum %d get %d", singleSum*numEnvs, sum)
	}
}

func TestReset(t *testing.T) {
	tree := NewBevTree()

	paral := NewParallelNode()

	tree.Root().SetChild(paral)

	rand.Seed(time.Now().Unix())

	lowUpdate, maxUpdate := 2, 10
	n := 10
	lowDepth, maxDepth := 5, 10
	for i := 0; i < n; i++ {
		ut := lowUpdate + rand.Intn(maxUpdate-lowUpdate+1)
		c := DecoratorNode(NewInverterNode())
		paral.AddChild(c)

		depth := 5 + rand.Intn(maxDepth-lowDepth)
		for d := 0; d < depth; d++ {
			cc := NewSucceederNode()
			c.SetChild(cc)
			c = cc
		}

		c.SetChild(NewBevNode(newBehaviorUpdate(ut)))
	}

	e := NewContext(nil)

	for i := 0; i < 100; i++ {
		tree.Update(e)
		tree.Stop(e)
	}
}
