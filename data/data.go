package data

// 数据上下文
type DataContext interface {
	Clear()
	Set(key string, val interface{})
	Val(key string) interface{}
}

// 黑板
type Blackboard struct {
	keyValues map[string]interface{}
}

func NewBlackboard() *Blackboard {
	return &Blackboard{
		keyValues: make(map[string]interface{}),
	}
}

func (bb *Blackboard) Set(key string, val interface{}) {
	bb.keyValues[key] = val
}

func (bb *Blackboard) Val(key string) interface{} {
	return bb.keyValues[key]
}

func (bb *Blackboard) Clear() {
	bb.keyValues = map[string]interface{}{}
}
