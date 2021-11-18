// +build debug

package bevtree

import (
	"reflect"
	"sync/atomic"

	"github.com/GodYY/gutils/assert"
)

var taskTotalGetTimes int64
var taskTotalPutTimes int64

func getTaskTotalGetTimes() int64 {
	return atomic.LoadInt64(&taskTotalGetTimes)
}

func getTaskTotalPutTimes() int64 {
	return atomic.LoadInt64(&taskTotalPutTimes)
}

func (p *taskPool) get() task {
	atomic.AddInt64(&taskTotalGetTimes, 1)
	return p.p.Get().(task)
}

func (p *taskPool) put(task task) {
	assert.Equal(reflect.TypeOf(task).Elem(), p.tt, "invalid task type")
	atomic.AddInt64(&taskTotalPutTimes, 1)
	p.p.Put(task)
}

var taskElemTotalGetTimes int64
var taskElemTotalPutTimes int64

func getTaskElemTotalGetTimes() int64 {
	return atomic.LoadInt64(&taskElemTotalGetTimes)
}

func getTaskElemTotalPutTimes() int64 {
	return atomic.LoadInt64(&taskElemTotalPutTimes)
}

func getTaskQueElem() *taskQueElem {
	atomic.AddInt64(&taskElemTotalGetTimes, 1)
	return taskElemPool.Get().(*taskQueElem)
}

func putTaskQueElem(e *taskQueElem) {
	assert.Assert(e != nil, "elem nil")
	atomic.AddInt64(&taskElemTotalPutTimes, 1)
	taskElemPool.Put(e)
}
