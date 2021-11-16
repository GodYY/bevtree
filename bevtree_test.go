package bevtree

import (
	"math/rand"
	"testing"
	"time"
)

type test struct {
	tree *BevTree
	e    *Env
}

func newTest() *test {
	t := &test{
		tree: NewTree(),
		e:    NewEnv(nil),
	}
	t.tree = NewTree()
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
		if t.e.DataCtx().Val(k) != v {
			tt.Fatalf("%s = %v(%v)", k, t.e.DataCtx().Val(k), v)
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

type bevFunc struct {
	f func(*Env) Result
}

func newBevFunc(f func(*Env) Result) *bevFunc {
	return &bevFunc{f: f}
}

func (b *bevFunc) OnInit(e *Env) {}

func (b *bevFunc) OnUpdate(e *Env) Result {
	// fmt.Println("behaviorFunc OnUpdate")
	return b.f(e)
}

func (b *bevFunc) OnTerminate(e *Env) {
}

type bevFuncDefiner struct {
	f func(e *Env) Result
}

func newBevFuncDefiner(f func(*Env) Result) *bevFuncDefiner {
	return &bevFuncDefiner{f: f}
}

func (d *bevFuncDefiner) CreateBev() Bev {
	return newBevFunc(d.f)
}

func (d *bevFuncDefiner) DestroyBev(Bev) {}

func TestRoot(t *testing.T) {
	test := newTest()
	test.run(t, RFailure, nil, 1)
}

func TestSequence(t *testing.T) {
	test := newTest()

	seq := NewSequence()

	test.tree.root().SetChild(seq)

	key := "counter"
	test.e.DataCtx().Set(key, 0)
	n := 2
	for i := 0; i < n; i++ {
		seq.AddChild(NewBev(newBevFuncDefiner(func(e *Env) Result {
			val := e.DataCtx().Val(key).(int) + 1
			e.DataCtx().Set(key, val)
			return RSuccess
		})))
	}

	test.run(t, RSuccess, map[string]interface{}{key: n}, 1)
}

func TestSelector(t *testing.T) {
	test := newTest()

	selc := NewSelector()

	test.tree.root().SetChild(selc)

	key := "selected"
	var selected int
	n := 10
	for i := 0; i < n; i++ {
		k := i
		selc.AddChild(NewBev(newBevFuncDefiner((func(e *Env) Result {
			if k == selected {
				test.e.DataCtx().Set(key, selected)
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

	seq := NewRandSequence()

	test.tree.root().SetChild(seq)

	key := "counter"
	test.e.DataCtx().Set(key, 0)
	n := 2
	for i := 0; i < n; i++ {
		k := i
		seq.AddChild(NewBev(newBevFuncDefiner(func(e *Env) Result {
			t.Log("seq", k, "update")
			val := e.DataCtx().Val(key).(int) + 1
			e.DataCtx().Set(key, val)
			return RSuccess
		})))
	}

	test.run(t, RSuccess, map[string]interface{}{key: n}, 1)
}

func TestRandomSelector(t *testing.T) {
	rand.Seed(time.Now().Unix())

	test := newTest()

	selc := NewRandSelector()

	test.tree.root().SetChild(selc)

	key := "selected"
	var selected int
	n := 10
	for i := 0; i < n; i++ {
		k := i
		selc.AddChild(NewBev(newBevFuncDefiner((func(e *Env) Result {
			t.Log("seq", k, "update")
			if k == selected {
				test.e.DataCtx().Set(key, selected)
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

	paral := NewParallel()

	test.tree.root().SetChild(paral)

	rand.Seed(time.Now().Unix())

	n := 2
	for i := 0; i < n; i++ {
		k := i + 1
		timer := time.NewTimer(1000 * time.Millisecond * time.Duration(k))
		paral.AddChild(NewBev(newBevFuncDefiner(func(e *Env) Result {
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

	paral := NewParallel()

	test.tree.root().SetChild(paral)

	rand.Seed(time.Now().Unix())

	lowUpdate, maxUpdate := 2, 10
	n := 10
	lowDepth, maxDepth := 5, 10
	for i := 0; i < n; i++ {
		k := i + 1
		ut := lowUpdate + rand.Intn(maxUpdate-lowUpdate+1)
		c := oneChildNode(NewInverter())
		paral.AddChild(c)

		depth := 5 + rand.Intn(maxDepth-lowDepth)
		for d := 0; d < depth; d++ {
			cc := NewSucceeder()
			c.SetChild(cc)
			c = cc
		}

		c.SetChild(NewBev(newBevFuncDefiner(func(e *Env) Result {
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
	repeater := NewRepeater(n)

	test.tree.root().SetChild(repeater)

	key := "counter"
	test.e.DataCtx().Set(key, 0)

	repeater.SetChild(NewBev(newBevFuncDefiner((func(e *Env) Result {
		t.Log("incr 1")
		e.DataCtx().Set(key, int(e.DataCtx().Val(key).(int))+1)
		return RSuccess
	}))))

	test.run(t, RSuccess, map[string]interface{}{key: n}, 1)
}

func TestInverter(t *testing.T) {
	test := newTest()

	inverter := NewInverter()

	test.tree.root().SetChild(inverter)

	inverter.SetChild(NewBev(newBevFuncDefiner(func(e *Env) Result {
		return RFailure
	})))

	test.run(t, RSuccess, nil, 1)
}

func TestSucceeder(t *testing.T) {
	test := newTest()

	succeeder := NewSucceeder()
	test.tree.root().SetChild(succeeder)

	succeeder.SetChild(NewBev(newBevFuncDefiner(func(e *Env) Result { return RFailure })))

	test.run(t, RSuccess, nil, 1)
}

func TestRepeatUntilFail(t *testing.T) {
	test := newTest()

	repeat := NewRepeatUntilFail(false)
	test.tree.root().SetChild(repeat)

	n := 4
	repeat.SetChild(NewBev(newBevFuncDefiner(func(e *Env) Result {
		t.Log("decr 1")

		n--

		if n <= 0 {
			return RFailure
		}

		return RSuccess
	})))

	test.run(t, RFailure, nil, 1)
}

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

func (b *behaviorIncr) OnInit(e *Env) {}

func (b *behaviorIncr) OnUpdate(e *Env) Result {
	if b.count >= b.limited {
		return RFailure
	}

	b.count++
	val := e.DataCtx().Val(b.key).(int)
	val++
	e.DataCtx().Set(b.key, val)
	if b.count >= b.limited {
		return RSuccess
	}

	return RRunning
}

func (b *behaviorIncr) OnTerminate(e *Env) { b.count = 0 }

type behaviorIncrDefiner struct {
	key     string
	limited int
}

func newBehaviorIncrDefiner(key string, limited int) *behaviorIncrDefiner {
	return &behaviorIncrDefiner{key: key, limited: limited}
}

func (d *behaviorIncrDefiner) CreateBev() Bev {
	return newBehaviorIncr(d.key, d.limited)
}

func (d *behaviorIncrDefiner) DestroyBev(Bev) {}

func TestShareTree(t *testing.T) {

	tree := NewTree()
	paral := NewParallel()
	tree.root().SetChild(paral)

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

		paral.AddChild(NewBev(newBehaviorIncrDefiner(key, limited)))
	}

	envs := make([]*Env, numEnvs)
	for i := 0; i < numEnvs; i++ {
		envs[i] = NewEnv(nil)
		envs[i].DataCtx().Set(key, 0)
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
		sum += envs[i].DataCtx().Val(key).(int)
	}

	if sum != singleSum*numEnvs {
		t.Fatalf("expected sum %d get %d", singleSum*numEnvs, sum)
	}
}

type behaviorUpdate struct {
	limited int
	count   int
}

func newBehaviorUpdate(limited int) *behaviorUpdate {
	return &behaviorUpdate{
		limited: limited,
	}
}

func (b *behaviorUpdate) OnInit(e *Env) {}

func (b *behaviorUpdate) OnUpdate(e *Env) Result {
	if b.count >= b.limited {
		return RSuccess
	}

	b.count++
	if b.count >= b.limited {
		return RSuccess
	}

	return RRunning
}

func (b *behaviorUpdate) OnTerminate(e *Env) {
	b.count = 0
}

type behaviorUpdateDefiner struct {
	limited int
}

func newBehaviorUpdateDefiner(limited int) *behaviorUpdateDefiner {
	return &behaviorUpdateDefiner{limited: limited}
}

func (d *behaviorUpdateDefiner) CreateBev() Bev {
	return newBehaviorUpdate(d.limited)
}

func (d *behaviorUpdateDefiner) DestroyBev(Bev) {

}
