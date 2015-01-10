package main

import (
	"container/ring"
)

// Ordered map.
type mapRing struct {
	m map[*playerConn]*ring.Ring // Maps conns to their position in the circular list.
	r *ring.Ring                 // Last added conn.
}

func newMapRing() *mapRing {
	return &mapRing{
		m: make(map[*playerConn]*ring.Ring),
	}
}

func (self *mapRing) Add(c *playerConn) {
	r := ring.New(1)
	r.Value = c

	if self.r != nil {
		self.r.Link(r)
	}
	self.r = r

	self.m[c] = r
}

func (self *mapRing) Remove(c *playerConn) {
	r := self.m[c]
	prev := r.Prev()

	prev.Unlink(1)

	delete(self.m, c)

	if len(self.m) == 0 {
		self.r = nil
	} else if self.r == r {
		self.r = prev
	}
}

func (self *mapRing) Next(c *playerConn) *playerConn {
	if r, ok := self.m[c]; ok {
		return r.Next().Value.(*playerConn)
	}
	return nil
}

//

type hub struct {
	// Registered connections.
	conns *mapRing

	// Channel used to register connections in the hub.
	regChan chan *playerConn
	// Channel used to unregister connections in the hub.
	unregChan chan *playerConn

	// Inbound messages from the connections.
	bcastChan chan []byte
}

func NewHub() *hub {
	h := &hub{
		conns:     newMapRing(),
		regChan:   make(chan *playerConn),
		unregChan: make(chan *playerConn),
		bcastChan: make(chan []byte),
	}

	go h.run()

	return h
}

func (h *hub) register(c *playerConn) {
	h.conns.Add(c)
}

func (h *hub) unregister(c *playerConn) {
	h.conns.Remove(c)

	close(c.hubToConn)
}

func (h *hub) sendBcast(m []byte) {
	for c := range h.conns.m {
		select {
		case c.hubToConn <- m:
		default:
			h.unregister(c)
		}
	}
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.regChan:
			h.register(c)
		case c := <-h.unregChan:
			h.unregister(c)
		case m := <-h.bcastChan:
			h.sendBcast(m)
		}
	}
}
