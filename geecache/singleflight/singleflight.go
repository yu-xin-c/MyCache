package singleflight

import "sync"

// call 是一个正在进行或已完成的 Do 调用
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group 表示一类工作，并形成一个命名空间，其中工作单元可以执行重复抑制。
type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call // lazily initialized
}

// Do 执行并返回给定函数的结果，确保对于给定键，一次只有一个执行正在进行。如果有重复进来，重复调用者等待原始完成并接收相同的结果。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
