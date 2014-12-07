package main

import (
	"log"

	"github.com/gorilla/websocket"
)

func main() {
	cli := websocket.DefaultDialer

	conn, _, err := cli.Dial("ws://localhost:3223/move/H7", nil)
	if err != nil {
		log.Println(err)
		return
	}
	// TODO: any need?
	defer conn.Close()

	var v interface{}
	for {
		err = conn.ReadJSON(&v)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(v)
	}
}
