package bevtree

import (
	"reflect"
	"sync"

	"github.com/GodYY/gutils/assert"
)

type taskPool struct {
	tt reflect.Type
	p  *sync.Pool
}

func newTaskPool(new func() Task) *taskPool {
	if debug {
		assert.Assert(new != nil, "new nil")
	}

	t := new()
	if debug {
		assert.Assert(t != nil, "new() nil task")
	}

	tt := reflect.TypeOf(t).Elem()

	p := &taskPool{
		tt: tt,
		p: &sync.Pool{
			New: func() interface{} {
				return new()
			},
		},
	}

	p.p.Put(t)

	return p
}

var taskElemPool = &sync.Pool{New: func() interface{} { return new(taskQueElem) }}
