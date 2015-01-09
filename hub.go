package main

import (
	"container/list"
)

// Ordered map.
type mapList struct {
	m map[*playerConn]*list.Element
	l *list.List
}

func newMapList() *mapList {
	return &mapList{
		m: make(map[*playerConn]*list.Element),
		l: list.New(),
	}
}

func (self *mapList) Add(c *playerConn) {
	self.m[c] = self.l.PushBack(c)
}

func (self *mapList) Remove(c *playerConn) {
	if elem, ok := self.m[c]; ok {
		self.l.Remove(elem)
		delete(self.m, c)
	}
}

//

type hub struct {
	// Registered connections.
	conns *mapList

	// Channel used to register connections in the hub.
	regChan chan *playerConn
	// Channel used to unregister connections in the hub.
	unregChan chan *playerConn

	// Inbound messages from the connections.
	bcastChan chan []byte
}

func NewHub() *hub {
	h := &hub{
		conns:     newMapList(),
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
