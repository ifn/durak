package main

import (
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/gorilla/websocket"
	"github.com/looplab/fsm"
)

type ErrMsg struct {
	Err string `json:"error"`
}

type DeskMsg struct {
	Desk [][]string `json:"desk"`
}

const (
	Start int = iota
	Move
)

type PlayerMsg struct {
	Cmd  int    `json:"command"`
	Card string `json:"card"`
}

//var CARD *regexp.Regexp = regexp.MustCompile(`[SCHD]([6-9JQKA]|10)`)
//
//func isValidCard(c string) bool {
//	return CARD.MatchString(c)
//}

type gameState struct {
	// player that should make current move
	p *websocket.Conn

	trump string

	fsm *fsm.FSM
}

var GSt *gameState

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
		case Start:
			self.gst.fsm.Event("start")
		case Move:
			self.gst.fsm.Event("move", m.Card)
		}
	}
}

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

func main() {
	http.HandleFunc("/", durakHandler)

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	GSt = &gameState{
		fsm: fsm.NewFSM(
			"collection",
			fsm.Events{
				{Name: "start", Src: []string{"collection"}, Dst: "distribution"},
				{Name: "move", Src: []string{"game"}, Dst: "game"},
				{Name: "move", Src: []string{"game"}, Dst: "distribution"},
				{Name: "move", Src: []string{"distribution"}, Dst: "game"},
			},
			fsm.Callbacks{},
		),
	}
}
