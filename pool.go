package bevtree

import (
	"sync"
)

type pool struct {
	p sync.Pool
}

func newPool(new func() interface{}) *pool {
	p := &pool{}
	p.p.New = new
	return p
}
