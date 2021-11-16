package bevtree

import (
	"reflect"
	"sync"

	"github.com/godyy/bevtree/internal/assert"
)

type taskPool struct {
	tt reflect.Type
	p  *sync.Pool
}

func newTaskPool(new func() task) *taskPool {
	if debug {
		assert.NotNilArg(new, "new")
	}

	t := new()
	if debug {
		assert.NotNil(t, "new() nil task")
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
