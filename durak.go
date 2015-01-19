package main

import (
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"

	"github.com/gorilla/websocket"
	sm "github.com/ifn/go-statemachine"
)

type DeskMsg struct {
	Desk [][]string `json:"desk"`
}

type PlayerMsg struct {
	Cmd  sm.EventType `json:"command"`
	Card string       `json:"card"`
}

//

var Order map[string]int = map[string]int{
	"6": 6, "7": 7, "8": 8, "9": 9, "10": 10,
	"J": 11,
	"Q": 12,
	"K": 13,
	"A": 14,
}

func higher(c0, c1, t string) int {
	// c0 and c1 have the same suit
	if c0[0] == c1[0] {
		if Order[c0[1:]] > Order[c1[1:]] {
			return 1
		}
		if Order[c0[1:]] < Order[c1[1:]] {
			return -1
		}
		return 0
	}
	// c0 is trump, c1 is not
	if c0[:1] == t {
		return 1
	}
	// c1 is trump, c0 is not
	if c1[:1] == t {
		return -1
	}
	// suits are different, both are not trump
	return -2
}

var CardRE *regexp.Regexp = regexp.MustCompile(`[SCHD]([6-9JQKA]|10)`)

func isValid(c string) bool {
	return CardRE.MatchString(c)
}

//

const (
	cmdStart sm.EventType = iota
	cmdMove

	cmdCount
)

const (
	stateCollection sm.State = iota

	stateAttack
	stateDefense

	stateCount
)

type roundResult bool

const (
	Beat roundResult = true
)

func stateToString(s sm.State) string {
	return [...]string{
		stateCollection: "COLLECTION",
		stateDefense:    "DEFENSE",
		stateAttack:     "ATTACK",
	}[s]
}

func cmdToString(t sm.EventType) string {
	return [...]string{
		cmdStart: "START",
		cmdMove:  "MOVE",
	}[t]
}

//

type cmdArgs struct {
	conn *playerConn
	card string
}

//

type gameState struct {
	// 1. fields that don't change during a game

	sm  *sm.StateMachine
	hub *hub

	// trump suit
	trump string

	// 2. fields that don't change during a round

	// attacker that started a round
	aconnStart *playerConn
	// defender
	dconn *playerConn

	// 3. fields that change during a round

	// attacker
	aconn *playerConn
	// card that should be beaten
	cardToBeat string
}

func (self *gameState) nextPlayer(c *playerConn) *playerConn {
	return self.hub.conns.Next(c)
}

func (self *gameState) finishRound(res roundResult) {
	switch res {
	case Beat:
		self.aconn = self.dconn
	case !Beat:
		self.aconn = self.nextPlayer(self.dconn)
	}
	self.dconn = self.nextPlayer(self.aconn)
}

//

func logOutOfTurn(pconn *playerConn) {
	log.Printf("out of turn: %v", pconn.conn.RemoteAddr())
}

func logWontBeat(c1, c2, t string) {
	log.Printf("%v won't bit %v, trump is ", c1, c2, t)
}

//

func NewGameState() *gameState {
	gst := new(gameState)

	gst.sm = sm.New(stateAttack, uint(stateCount), uint(cmdCount))

	gst.sm.On(cmdMove,
		[]sm.State{stateAttack},
		gst.handleMoveInAttack,
	)
	gst.sm.On(cmdMove,
		[]sm.State{stateDefense},
		gst.handleMoveInDefense,
	)

	gst.sm.On(cmdMove,
		[]sm.State{stateAttack, stateDefense},
		gst.showDesk,
	)

	gst.hub = NewHub()

	return gst
}

// event handlers

func (self *gameState) showDesk(s sm.State, e *sm.Event) sm.State {
	log.Println("showDesk")

	return s
}

func (self *gameState) handleMoveInAttack(s sm.State, e *sm.Event) sm.State {
	conn := e.Data.(cmdArgs).conn
	card := e.Data.(cmdArgs).card

	// check that it's conn's turn to move
	if conn != self.aconn {
		logOutOfTurn(conn)
		return s
	}

	// attacker sent the card
	if card != "" {
		self.cardToBeat = card
		return stateDefense
	}

	// attacker sent no card

	aconn := self.nextPlayer(self.aconn)
	if aconn == self.dconn {
		aconn = self.nextPlayer(aconn)
	}

	// check if all attackers have been polled
	if aconn == self.aconnStart {
		self.finishRound(Beat)
		return stateAttack
	}

	self.aconn = aconn
	return stateAttack
}

func (self *gameState) handleMoveInDefense(s sm.State, e *sm.Event) sm.State {
	conn := e.Data.(cmdArgs).conn
	card := e.Data.(cmdArgs).card

	// check that it's conn's turn to move
	if conn != self.dconn {
		logOutOfTurn(conn)
		return s
	}

	// defender takes the cards
	if card == "" {
		self.finishRound(!Beat)
		return stateAttack
	}

	// check that the sent card is capable to beat
	if higher(card, self.cardToBeat, self.trump) != 1 {
		logWontBeat(card, self.cardToBeat, self.trump)
		return s
	}

	return stateAttack
}

//

type playerConn struct {
	gst *gameState

	conn      *websocket.Conn
	hubToConn chan []byte
}

func (self *playerConn) write() {
	defer func() {
		err := self.conn.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	for {
		select {
		case m := <-self.hubToConn:
			//TODO: text or binary?
			err := self.conn.WriteMessage(websocket.TextMessage, m)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func (self *playerConn) read() {
	defer func() {
		err := self.conn.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	var m PlayerMsg

	for {
		err := self.conn.ReadJSON(&m)
		if err != nil {
			log.Println(err)
			return
		}

		switch m.Cmd {
		case cmdStart:
			log.Println(cmdToString(cmdStart))
		case cmdMove:
			event := &sm.Event{cmdMove, cmdArgs{self, m.Card}}

			err = self.gst.sm.Emit(event)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

//

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func playerHandler(gst *gameState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		p := &playerConn{gst, conn, make(chan []byte)}

		gst.hub.regChan <- p
		defer func() {
			gst.hub.unregChan <- p
		}()

		go p.write()
		p.read()
	}
}

//

func startDurakSrv() error {
	gst := NewGameState()

	http.HandleFunc("/", playerHandler(gst))

	return http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}

func main() {
	err := startDurakSrv()
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}
