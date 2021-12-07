package bevtree

import (
	"sync"
)

// A packaging of go built-in sync.Pool.
type pool struct {
	p sync.Pool
}

func newPool(new func() interface{}) *pool {
	p := &pool{}
	p.p.New = new
	return p
}
