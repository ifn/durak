package main

type hub struct {
	// Registered connections.
	conns map[*playerConn]int
	// Registered connections' order.
	connOrder []*playerConn

	// Channel used to register connections in the hub.
	regChan chan *playerConn
	// Channel used to unregister connections in the hub.
	unregChan chan *playerConn

	// Inbound messages from the connections.
	bcastChan chan []byte
}

func NewHub() *hub {
	h := &hub{
		conns:     make(map[*playerConn]int),
		regChan:   make(chan *playerConn),
		unregChan: make(chan *playerConn),
		bcastChan: make(chan []byte),
	}

	go h.run()

	return h
}

func (h *hub) register(c *playerConn) {
	h.conns[c] = len(h.connOrder)

	h.connOrder = append(h.connOrder, c)
}

func (h *hub) unregister(c *playerConn) {
	pos := h.conns[c]
	h.connOrder[pos] = nil

	delete(h.conns, c)

	close(c.hubToConn)
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.regChan:
			h.register(c)
		case c := <-h.unregChan:
			if _, ok := h.conns[c]; ok {
				h.unregister(c)
			}
		case m := <-h.bcastChan:
			for c := range h.conns {
				select {
				case c.hubToConn <- m:
				default:
					h.unregister(c)
				}
			}
		}
	}
}
