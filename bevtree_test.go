package bevtree

import (
	"fmt"
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
		e: NewEnv(nil),
	}
	t.tree = NewTree()
	return t
}

func (t *test) run(tt *testing.T, expectedResult Result, expectedKeyValues map[string]interface{}, tick int) {
	result := RRunning

	for i := 0; i < tick; i++ {
		fmt.Println("run", i, "start")
		result = RRunning
		for result == RRunning {
			fmt.Println("update")
			result = t.tree.Update(t.e)
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Println("run", i, "end", result)
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

type behaviorFunc struct {
	f func(e *Env) Result
}

func newFunc(f func(e *Env) Result) *behaviorFunc {
	return &behaviorFunc{f: f}
}

func (b *behaviorFunc) OnStart(e *Env) {
	// fmt.Println("behaviorFunc OnStart")
}

func (b *behaviorFunc) OnUpdate(e *Env) Result {
	// fmt.Println("behaviorFunc OnUpdate")
	return b.f(e)
}

func (b *behaviorFunc) OnEnd(e *Env) {
	// fmt.Println("behaviorFunc OnEnd")
}

func (b *behaviorFunc) OnStop(e *Env) {
	// fmt.Println("behaviorFunc OnStop")
}

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
		seq.AddChild(NewBehavior(newFunc(func(e *Env) Result {
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
		selc.AddChild(NewBehavior(newFunc((func(e *Env) Result {
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
		seq.AddChild(NewBehavior(newFunc(func(e *Env) Result {
			fmt.Println("seq", k, "update")
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
		selc.AddChild(NewBehavior(newFunc((func(e *Env) Result {
			fmt.Println("seq", k, "update")
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

	test.run(t, RSuccess, map[string]interface{}{key: selected}, 5)
}

func TestParallel(t *testing.T) {
	test := newTest()

	paral := NewParallel()

	test.tree.root().SetChild(paral)

	rand.Seed(time.Now().Unix())

	n := 2
	for i := 0; i < n; i++ {
		k := i + 1
		timer := time.NewTimer(time.Millisecond * time.Duration(k))
		paral.AddChild(NewBehavior(newFunc(func(e *Env) Result {
			select {
			case <-timer.C:
				t.Logf("timer No.%d up", k)
				fmt.Printf("timer No.%d up\n", k)
				return RSuccess
			default:
				t.Logf("timer No.%d update", k)
				fmt.Printf("timer No.%d update\n", k)
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

	n := 2
	for i := 0; i < n; i++ {
		k := i + 1
		timer := time.NewTimer(time.Second * time.Duration((2-k+1)*2))
		paral.AddChild(NewBehavior(newFunc(func(e *Env) Result {
			select {
			case <-timer.C:
				t.Logf("timer No.%d up", k)
				fmt.Printf("timer No.%d up\n", k)
				return RFailure
			default:
				t.Logf("timer No.%d update", k)
				fmt.Printf("timer No.%d update\n", k)
				return RRunning
			}
		})))
	}

	test.run(t, RSuccess, nil, 2)
}

func TestRepeater(t *testing.T) {
	test := newTest()

	n := 10
	repeater := NewRepeater(n)

	test.tree.root().SetChild(repeater)

	key := "counter"
	test.e.DataCtx().Set(key, 0)

	repeater.SetChild(NewBehavior(newFunc((func(e *Env) Result {
		e.DataCtx().Set(key, int(e.DataCtx().Val(key).(int))+1)
		return RSuccess
	}))))

	test.run(t, RSuccess, map[string]interface{}{key: n}, 1)
}

func TestInverter(t *testing.T) {
	test := newTest()

	inverter := NewInverter()

	test.tree.root().SetChild(inverter)

	inverter.SetChild(NewBehavior(newFunc(func(e *Env) Result {
		return RFailure
	})))

	test.run(t, RSuccess, nil, 1)
}

func TestSucceeder(t *testing.T) {
	test := newTest()

	succeeder := NewSucceeder()
	test.tree.root().SetChild(succeeder)

	succeeder.SetChild(NewBehavior(newFunc(func(e *Env) Result { return RFailure })))

	test.run(t, RSuccess, nil, 1)
}

func TestRepeatUntilFail(t *testing.T) {
	test := newTest()

	repeat := NewRepeatUntilFail()
	test.tree.root().SetChild(repeat)

	n := 4
	repeat.SetChild(NewBehavior(newFunc(func(e *Env) Result {
		n--

		if n <= 0 {
			return RFailure
		}

		return RSuccess
	})))

	test.run(t, RFailure, nil, 1)
}
