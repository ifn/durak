package main

import (
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"

	"github.com/gorilla/websocket"
	sm "github.com/tchap/go-statemachine"
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
	stateDistribution
	stateGame

	stateAttack
	stateDefense

	stateCount
)

func stateToString(s sm.State) string {
	return [...]string{
		stateCollection:   "COLLECTION",
		stateDistribution: "DISTRIBUTION",
		stateGame:         "GAME",
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
	conn *websocket.Conn
	card string
}

//

type gameState struct {
	// trump suit
	trump string

	// attacker
	aconn *websocket.Conn
	// defender
	dconn *websocket.Conn
	// card that should be beaten
	cardToBeat string

	sm *sm.StateMachine
}

func (self *gameState) nextAttacker() *websocket.Conn {
	return nil
}

func (self *gameState) nextDefender() *websocket.Conn {
	return nil
}

func logOutOfTurn(conn *websocket.Conn) {
	log.Printf("out of turn: %v", conn)
}

func logWontBeat(c1, c2, t string) {
	log.Printf("%v won't bit %v, trump is ", c1, c2, t)
}

func NewGameState() *gameState {
	gst := new(gameState)

	gst.sm = sm.New(stateGame, uint(stateCount), uint(cmdCount))

	gst.sm.On(cmdMove,
		[]sm.State{stateAttack},
		gst.handleMoveInAttack,
	)

	gst.sm.On(cmdMove,
		[]sm.State{stateDefense},
		gst.handleMoveInDefense,
	)

	return gst
}

var GSt *gameState = NewGameState()

// event handlers

func (self *gameState) handleMoveInAttack(s sm.State, e *sm.Event) sm.State {
	conn := e.Data.(cmdArgs).conn
	card := e.Data.(cmdArgs).card

	if conn != self.aconn {
		logOutOfTurn(conn)
		return s
	}

	// attacker won't push more cards
	if card == "" {
		self.aconn = self.nextAttacker()
		return s
	}

	self.cardToBeat = card

	return stateDefense
}

func (self *gameState) handleMoveInDefense(s sm.State, e *sm.Event) sm.State {
	conn := e.Data.(cmdArgs).conn
	card := e.Data.(cmdArgs).card

	if conn != self.dconn {
		logOutOfTurn(conn)
		return s
	}

	// defender takes the cards
	if card == "" {
		self.aconn = self.nextAttacker()
		self.dconn = self.nextDefender()

		//distribute cards

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

type durakSrv struct {
	conn *websocket.Conn
	gst  *gameState
}

func (self *durakSrv) read() {
	var m PlayerMsg

	for {
		err := self.conn.ReadJSON(&m)
		if err != nil {
			log.Println(err)
			return
		}

		switch m.Cmd {
		case cmdStart:
		case cmdMove:
			event := &sm.Event{cmdMove, cmdArgs{self.conn, m.Card}}

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

func durakHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close() //dbg

	d := durakSrv{conn, GSt}

	d.read()
}

//

func main() {
	http.HandleFunc("/", durakHandler)

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}
