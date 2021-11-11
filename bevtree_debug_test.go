// +build debug

package bevtree

import (
	"math/rand"
	"testing"
	"time"
)

func TestTaskPool(t *testing.T) {
	test := newTest()

	paral := NewParallel()

	test.tree.root().SetChild(paral)

	rand.Seed(time.Now().Unix())

	lowUpdate, maxUpdate := 2, 10
	n := 10
	lowDepth, maxDepth := 5, 10
	for i := 0; i < n; i++ {
		ut := lowUpdate + rand.Intn(maxUpdate-lowUpdate+1)
		c := oneChildNode(NewInverter())
		paral.AddChild(c)

		depth := 5 + rand.Intn(maxDepth-lowDepth)
		for d := 0; d < depth; d++ {
			cc := NewSucceeder()
			c.SetChild(cc)
			c = cc
		}

		c.SetChild(NewBev(newBehaviorUpdateDefiner(ut)))
	}

	test.run(t, RFailure, nil, 50)
	test.close()

	if getTaskTotalGetTimes() != getTaskTotalPutTimes() {
		t.Fatalf("taskTotalGetTimes(%d) != taskTotalPutTimes(%d), memory leak", getTaskTotalGetTimes(), getTaskTotalPutTimes())
	} else {
		t.Logf("taskTotalGetTimes(%d) == taskTotalPutTimes(%d)", getTaskTotalGetTimes(), getTaskTotalPutTimes())
	}

	if getTaskElemTotalGetTimes() != getTaskElemTotalPutTimes() {
		t.Fatalf("taskElemTotalGetTimes(%d) != taskElemTotalPutTimes(%d), memory leak", getTaskElemTotalGetTimes(), getTaskElemTotalPutTimes())
	} else {
		t.Logf("taskElemTotalGetTimes(%d) == taskElemTotalPutTimes(%d)", getTaskElemTotalGetTimes(), getTaskElemTotalPutTimes())
	}
}

func TestReset(t *testing.T) {
	tree := NewTree()

	paral := NewParallel()

	tree.root().SetChild(paral)

	rand.Seed(time.Now().Unix())

	lowUpdate, maxUpdate := 2, 10
	n := 10
	lowDepth, maxDepth := 5, 10
	for i := 0; i < n; i++ {
		ut := lowUpdate + rand.Intn(maxUpdate-lowUpdate+1)
		c := oneChildNode(NewInverter())
		paral.AddChild(c)

		depth := 5 + rand.Intn(maxDepth-lowDepth)
		for d := 0; d < depth; d++ {
			cc := NewSucceeder()
			c.SetChild(cc)
			c = cc
		}

		c.SetChild(NewBev(newBehaviorUpdateDefiner(ut)))
	}

	e := NewEnv(nil)

	for i := 0; i < 100; i++ {
		tree.Update(e)
		tree.Reset(e)

		if getTaskTotalGetTimes() != getTaskTotalPutTimes() {
			t.Fatalf("No.%d taskTotalGetTimes(%d) != taskTotalPutTimes(%d), memory leak", i, getTaskTotalGetTimes(), getTaskTotalPutTimes())
		} else {
			t.Logf("No.%d taskTotalGetTimes(%d) == taskTotalPutTimes(%d)", i, getTaskTotalGetTimes(), getTaskTotalPutTimes())
		}

		if getTaskElemTotalGetTimes() != getTaskElemTotalPutTimes() {
			t.Fatalf("No.%d taskElemTotalGetTimes(%d) != taskElemTotalPutTimes(%d), memory leak", i, getTaskElemTotalGetTimes(), getTaskElemTotalPutTimes())
		} else {
			t.Logf("No.%d taskElemTotalGetTimes(%d) == taskElemTotalPutTimes(%d)", i, getTaskElemTotalGetTimes(), getTaskElemTotalPutTimes())
		}
	}
}
