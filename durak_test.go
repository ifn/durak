package main

import (
	"os"
	"testing"

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

//

func TestMapRing1(t *testing.T) {
	mr := newMapRing()
	pc := new(playerConn)

	if len(mr.m) != 0 ||
		mr.r != nil {
		t.Fail()
	}

	mr.Add(pc)

	r := mr.m[pc]

	if len(mr.m) != 1 || mr.r.Len() != 1 ||
		mr.r != r {
		t.Fail()
	}

	mr.Remove(pc)

	if len(mr.m) != 0 ||
		mr.r != nil {
		t.Fail()
	}
}

func TestMapRing2(t *testing.T) {
	mr := newMapRing()
	pc1 := new(playerConn)
	pc2 := new(playerConn)

	mr.Add(pc1)
	mr.Add(pc2)

	r1 := mr.m[pc1]
	r2 := mr.m[pc2]

	if len(mr.m) != 2 || mr.r.Len() != 2 ||
		r1.Next() != r2 || r2.Next() != r1 ||
		mr.r != r2 {
		t.Fail()
	}

	mr.Remove(pc2)

	r1 = mr.m[pc1]

	if len(mr.m) != 1 || mr.r.Len() != 1 ||
		r1.Next() != r1 ||
		mr.r != r1 {
		t.Fail()
	}
}

//

func TestDurak(t *testing.T) {
	cli := websocket.DefaultDialer

	conn, _, err := cli.Dial("ws://localhost:"+os.Getenv("PORT"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close() //dbg

	m := PlayerMsg{cmdMove, "C9"}
	//m := PlayerMsg{Cmd: cmdStart}

	err = conn.WriteJSON(m)
	if err != nil {
		t.Fatal(err)
	}
}
