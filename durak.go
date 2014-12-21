package main

import (
	"log"
	"net/http"
	"os"
	//"regexp"
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

type CmdType int

const (
	Start CmdType = iota
	Move
)

type PlayerMsg struct {
	Cmd  CmdType `json:"command"`
	Card string  `json:"card"`
}

//var CARD *regexp.Regexp = regexp.MustCompile(`[SCHD]([6-9JQKA]|10)`)
//
//func isValidCard(c string) bool {
//	return CARD.MatchString(c)
//}

type durakSrv struct {
	conn *websocket.Conn
	fsm  *fsm.FSM
}

func NewDurakSrv(conn *websocket.Conn) *durakSrv {
	fsm := fsm.NewFSM(
		"collection",
		fsm.Events{
			{Name: "start", Src: []string{"collection"}, Dst: "distribution"},
			{Name: "move", Src: []string{"game"}, Dst: "game"},
			{Name: "move", Src: []string{"game"}, Dst: "distribution"},
			{Name: "move", Src: []string{"distribution"}, Dst: "game"},
		},
		fsm.Callbacks{},
	)

	d := durakSrv{conn, fsm}
	return &d
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
			self.fsm.Event("start")
		case Move:
			self.fsm.Event("move", m.Card)
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

	d := NewDurakSrv(conn)
	d.read()
}

func main() {
	http.HandleFunc("/", durakHandler)

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}
