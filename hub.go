package main

type collection interface {
	Add(interface{})
	Remove(interface{})
	Enumerate() <-chan interface{}
}

type hub struct {
	// Registered connections.
	conns collection

	// Channel used to register connections in the hub.
	regChan chan *playerConn
	// Channel used to unregister connections in the hub.
	unregChan chan *playerConn

	// Inbound messages from the connections.
	bcastChan chan []byte
}

func NewHub() *hub {
	h := &hub{
		conns:     newMapRing(), //TODO: hub should know nothing about mapRing
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
	for c := range h.conns.Enumerate() {
		c := c.(*playerConn)

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
