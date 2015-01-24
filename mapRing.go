package main

import (
	"container/ring"
)

type mapRing struct {
	m map[interface{}]*ring.Ring // Maps elements to their position in the circular list.
	r *ring.Ring                 // The youngest element.
}

func newMapRing() *mapRing {
	return &mapRing{
		m: make(map[interface{}]*ring.Ring),
	}
}

func (self *mapRing) Add(e interface{}) {
	r := ring.New(1)
	r.Value = e

	if self.r != nil {
		self.r.Link(r)
	}
	self.r = r

	self.m[e] = r
}

func (self *mapRing) Remove(e interface{}) {
	r, ok := self.m[e]
	if !ok {
		return
	}
	prev := r.Prev()

	prev.Unlink(1)

	delete(self.m, e)

	if len(self.m) == 0 {
		self.r = nil
	} else if self.r == r {
		self.r = prev
	}
}

func (self *mapRing) Enumerate() <-chan interface{} {
	n := len(self.m)
	ch := make(chan interface{}, n)

	go func() {
		e := self.r
		for i := 0; i < n; i++ {
			e = e.Next()
			ch <- e.Value
		}
		close(ch)
	}()

	return ch
}

func (self *mapRing) Next(e interface{}) interface{} {
	if r, ok := self.m[e]; ok {
		return r.Next().Value
	}
	return nil
}
