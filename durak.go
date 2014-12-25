package main

import (
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/gorilla/websocket"
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

type gameState struct {
	// player that should make a current move
	p *websocket.Conn

	trump string
}

var GSt *gameState = &gameState{}

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
			log.Println(Start)
		case Move:
			log.Println(Move, m.Card)
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
}
