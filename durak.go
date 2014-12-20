package main

import (
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/gorilla/websocket"
)

type ErrMsg struct {
	Err string `json:"error"`
}

type DeskMsg struct {
	Desk [][]string `json:"desk"`
}

type MoveMsg struct {
	Card string `json:"card"`
}

var CARD *regexp.Regexp = regexp.MustCompile(`[SCHD]([6-9JQKA]|10)`)

func isValidCard(c string) bool {
	return CARD.MatchString(c)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close() //dbg

	var m MoveMsg

	for {
		err = conn.ReadJSON(&m)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println(m.Card)
		log.Println(isValidCard(m.Card))
	}
}

func main() {
	http.HandleFunc("/", handler)
	
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
