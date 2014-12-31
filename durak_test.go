package main

import (
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHigher(t *testing.T) {
	r := higher("S7", "S6", "")
	if r != 1 {
		t.Fail()
	}
	r = higher("S6", "S7", "")
	if r != -1 {
		t.Fail()
	}
	r = higher("S6", "S6", "")
	if r != 0 {
		t.Fail()
	}
	r = higher("S6", "C7", "S")
	if r != 1 {
		t.Fail()
	}
	r = higher("S7", "C6", "C")
	if r != -1 {
		t.Fail()
	}
	r = higher("S7", "C6", "H")
	if r != -2 {
		t.Fail()
	}
}

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
