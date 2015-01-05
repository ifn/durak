package main

// FIXME: bad name
type Sender interface {
	// Channel used to send outbound messages from hub to connections.
	GetChan() chan<- []byte
}

type hub struct {
	// Registered connections.
	connections map[Sender]bool

	// Channel used to register connections in hub.
	register chan Sender
	// Channel used to unregister connections in hub.
	unregister chan Sender

	// Inbound messages from the connections.
	broadcast chan []byte
}

func NewHub() *hub {
	h := &hub{
		connections: make(map[Sender]bool),
		register:    make(chan Sender),
		unregister:  make(chan Sender),
		broadcast:   make(chan []byte),
	}

	go h.run()

	return h
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				delete(h.connections, c)
				close(c.GetChan())
			}
		case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.GetChan() <- m:
				default:
					delete(h.connections, c)
					close(c.GetChan())
				}
			}
		}
	}
}
