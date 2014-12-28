package main

import (
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestDurak(t *testing.T) {
	cli := websocket.DefaultDialer

	conn, _, err := cli.Dial("ws://localhost:"+os.Getenv("PORT"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close() //dbg

	m := PlayerMsg{cmdMove, "S7"}

	n := 2
	for i := 0; i < n; i++ {
		err = conn.WriteJSON(m)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(250 * time.Millisecond)
	}
}
