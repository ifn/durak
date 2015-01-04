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

func TestIsValid(t *testing.T) {
	if !isValid("S7") || !isValid("H10") || !isValid("DQ") {
		t.Fail()
	}
}

func TestDurakMove(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	go func() {
		err := startDurakSrv()
		if err != nil {
			t.Log(err)
		}
	}()

	cli := websocket.DefaultDialer

	conn, _, err := cli.Dial("ws://localhost:"+os.Getenv("PORT"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close() //dbg

	time.Sleep(time.Millisecond * 100)
	GSt.aconn = GSt.hub[0]

	GSt.sm.SetState(stateAttack)

	m := PlayerMsg{cmdMove, "C9"}

	err = conn.WriteJSON(m)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)
	state := GSt.sm.GetState()
	if state != stateDefense {
		t.Fatalf("%v, expected %v", stateToString(state), stateToString(stateDefense))
	}
}

func TestDurak(t *testing.T) {
	cli := websocket.DefaultDialer

	conn, _, err := cli.Dial("ws://localhost:"+os.Getenv("PORT"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close() //dbg

	m := PlayerMsg{cmdMove, "C9"}

	err = conn.WriteJSON(m)
	if err != nil {
		t.Fatal(err)
	}
}
