package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/manuelpepe/tincho/internal/tincho"
)

var addr = flag.String("addr", "localhost:5555", "http service address")
var roomID = flag.String("room", "", "room id")
var user = flag.String("user", "", "username")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	if roomID == nil || user == nil || *roomID == "" || *user == "" {
		log.Fatal("room and user are required")
	}

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/join", RawQuery: "room=" + *roomID + "&player=" + *user}
	log.Printf("connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	// read stdin into channel
	commands := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			commands <- scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			log.Println(err)
		}
	}()

	// read from websocket into channel
	done := make(chan struct{})
	recieved := make(chan any)
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			var update tincho.Update
			if err := json.Unmarshal(message, &update); err != nil {
				log.Println(err)
				return
			}
			recieved <- update
		}
	}()

	for {
		select {
		case <-done:
			log.Println("bye")
			return
		case cmd := <-commands:
			err := conn.WriteMessage(websocket.TextMessage, parse(cmd))
			if err != nil {
				log.Println("write:", err)
				return
			}
			log.Println("sent: ", cmd)
		case rcvd := <-recieved:
			log.Printf("recieved: %+v\n", rcvd)
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func parse(cmd string) []byte {
	var action tincho.Action
	var data any
	switch cmd {
	case "start":
		action = tincho.Action{Type: tincho.ActionStart}
		data = nil
	case "draw":
		action = tincho.Action{Type: tincho.ActionDraw}
		data = tincho.DrawAction{Source: tincho.DrawSourcePile}
	case "disc 0":
		action = tincho.Action{Type: tincho.ActionDiscard}
		data = tincho.DiscardAction{CardPosition: 0}
	case "disc 1":
		action = tincho.Action{Type: tincho.ActionDiscard}
		data = tincho.DiscardAction{CardPosition: 1}
	case "disc 2":
		action = tincho.Action{Type: tincho.ActionDiscard}
		data = tincho.DiscardAction{CardPosition: 2}
	case "disc 3":
		action = tincho.Action{Type: tincho.ActionDiscard}
		data = tincho.DiscardAction{CardPosition: 3}
	default:
		return nil
	}
	var err error
	action.Data, err = json.Marshal(data)
	if err != nil {
		log.Println("error marshalling data: ", err)
		return nil
	}
	msg, err := json.Marshal(action)
	if err != nil {
		log.Println("error marshalling action: ", err)
		return nil
	}
	return msg

}
