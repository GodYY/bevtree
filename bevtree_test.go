package bevtree

import (
	"math/rand"
	"testing"
	"time"
)

var (
	btFunc   = RegisterBevType("func", func() Bev { return new(bevFunc) }, func() BevParams { return new(bevFuncParams) })
	btIncr   = RegisterBevType("incr", func() Bev { return new(behaviorIncr) }, func() BevParams { return new(behaviorIncrParams) })
	btUpdate = RegisterBevType("update", func() Bev { return new(behaviorUpdate) }, func() BevParams { return new(behaviorUpdateParams) })
)

type bevFuncParams struct {
	f func(*Context) Result
}

func newBevFuncParams(f func(*Context) Result) *bevFuncParams {
	return &bevFuncParams{f: f}
}

func (bevFuncParams) BevType() BevType { return btFunc }

type bevFunc struct {
	*bevFuncParams
}

func (b *bevFunc) BevType() BevType { return btFunc }

func (b *bevFunc) OnCreate(params BevParams) {
	b.bevFuncParams = params.((*bevFuncParams))
}

func (b *bevFunc) OnDestroy() {}

func (b *bevFunc) OnInit(e *Context) bool { return true }

func (b *bevFunc) OnUpdate(e *Context) Result {
	// fmt.Println("behaviorFunc OnUpdate")
	return b.f(e)
}

func (b *bevFunc) OnTerminate(e *Context) {
}

type behaviorIncrParams struct {
	key     string
	limited int
}

func newBehaviorIncrParams(key string, limited int) *behaviorIncrParams {
	return &behaviorIncrParams{key: key, limited: limited}
}

func (behaviorIncrParams) BevType() BevType { return btIncr }

type behaviorIncr struct {
	*behaviorIncrParams
	count int
}

func (b *behaviorIncr) BevType() BevType { return btIncr }
func (b *behaviorIncr) OnCreate(params BevParams) {
	b.behaviorIncrParams = params.(*behaviorIncrParams)
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

func (b *behaviorIncr) OnTerminate(e *Context) { b.count = 0 }

type behaviorUpdateParams struct {
	limited int
}

func newBehaviorUpdateParams(lmited int) *behaviorUpdateParams {
	return &behaviorUpdateParams{limited: lmited}
}

func (behaviorUpdateParams) BevType() BevType { return btUpdate }

type behaviorUpdate struct {
	*behaviorUpdateParams
	count int
}

func (b *behaviorUpdate) BevType() BevType { return btUpdate }

func (b *behaviorUpdate) OnCreate(params BevParams) {
	b.behaviorUpdateParams = params.(*behaviorUpdateParams)
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

func (b *behaviorUpdate) OnTerminate(e *Context) { b.count = 0 }

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
		seq.AddChild(NewBevNode(newBevFuncParams(func(e *Context) Result {
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
		selc.AddChild(NewBevNode(newBevFuncParams((func(e *Context) Result {
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
		seq.AddChild(NewBevNode(newBevFuncParams(func(e *Context) Result {
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
		selc.AddChild(NewBevNode(newBevFuncParams((func(e *Context) Result {
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
		paral.AddChild(NewBevNode(newBevFuncParams(func(e *Context) Result {
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

		c.SetChild(NewBevNode(newBevFuncParams(func(e *Context) Result {
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

	repeater.SetChild(NewBevNode(newBevFuncParams((func(e *Context) Result {
		e.IncInt(key)
		return RSuccess
	}))))

	test.run(t, RSuccess, map[string]interface{}{key: n}, 1)
}

func TestInverter(t *testing.T) {
	test := newTest()

	inverter := NewInverterNode()

	test.tree.Root().SetChild(inverter)

	inverter.SetChild(NewBevNode(newBevFuncParams(func(e *Context) Result {
		return RFailure
	})))

	test.run(t, RSuccess, nil, 1)
}

func TestSucceeder(t *testing.T) {
	test := newTest()

	succeeder := NewSucceederNode()
	test.tree.Root().SetChild(succeeder)

	succeeder.SetChild(NewBevNode(newBevFuncParams(func(e *Context) Result { return RFailure })))

	test.run(t, RSuccess, nil, 1)
}

func TestRepeatUntilFail(t *testing.T) {
	test := newTest()

	repeat := NewRepeatUntilFailNode(false)
	test.tree.Root().SetChild(repeat)

	n := 4
	repeat.SetChild(NewBevNode(newBevFuncParams(func(e *Context) Result {
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

		paral.AddChild(NewBevNode(newBehaviorIncrParams(key, limited)))
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

		c.SetChild(NewBevNode(newBehaviorUpdateParams(ut)))
	}

	e := NewContext(nil)

	for i := 0; i < 100; i++ {
		tree.Update(e)
		tree.Stop(e)
	}
}

func TestRemoveChild(t *testing.T) {
	key := "key"
	sum := 0

	unit := 1

	rand.Seed(time.Now().UnixNano())

	tree := NewBevTree()
	paral := NewParallelNode()
	tree.Root().SetChild(paral)

	bd := newBevBBIncrParams(key, unit)

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

	for paral.ChildCount() > 0 {
		paral.RemoveChild(0)
	}

	if paral.ChildCount() > 0 {
		t.FailNow()
	}

	context := NewContext(nil)
	if tree.Update(context) != RFailure {
		t.FailNow()
	}
}
