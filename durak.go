package main

import (
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/gorilla/websocket"
	sm "github.com/tchap/go-statemachine"
)

type ErrMsg struct {
	Err string `json:"error"`
}

type DeskMsg struct {
	Desk [][]string `json:"desk"`
}

type PlayerMsg struct {
	Cmd  sm.EventType `json:"command"`
	Card string       `json:"card"`
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
	// player that should make the move now
	p *websocket.Conn

	// card that should be beaten
	card string

	trump string

	sm *sm.StateMachine
}

func NewGameState() *gameState {
	gst := new(gameState)

	gst.sm = sm.New(stateGame, uint(stateCount), uint(cmdCount))

	gst.sm.On(cmdMove,
		[]sm.State{stateGame},
		gst.handleMove)

	return gst
}

var GSt *gameState = NewGameState()

// event handlers

func (self *gameState) handleMove(s sm.State, e *sm.Event) sm.State {
	conn := e.Data.(cmdArgs).conn
	card := e.Data.(cmdArgs).card

	if conn != self.p {
		log.Printf("it's %v's turn to make a move", self.p)
		return s
	}

	log.Println(card)

	return s
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
