package main

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestDurak(*testing.T) {
	cli := websocket.DefaultDialer

	conn, _, err := cli.Dial("ws://localhost:"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close() //dbg

	m := MoveMsg{"S7"}

	n := 10
	for i := 0; i < n; i++ {
		err = conn.WriteJSON(m)
		if err != nil {
			log.Println(err)
			return
		}

		time.Sleep(500 * time.Millisecond)
	}
}
