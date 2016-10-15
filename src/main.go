package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Client struct {
	hub *Hub

	conn *websocket.Conn

	send chan []byte //buffered chan

	name string
}

type Hub struct {
	clients map[*Client]bool // registered clients

	broadcast chan []byte

	register chan *Client

	unregister chan *Client

	write chan []byte
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		write:      make(chan []byte),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			// fmt.Println("---------ACTIVE--------------")
			// for client := range h.clients {
			// 	fmt.Println(client.name)
			// }
			// fmt.Println("-----------------------")
		case client := <-h.unregister:
			// 	fmt.Println("second")
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.conn.Close()
				// close(client.send)
			}
			// fmt.Println("---------ACTIVE--------------")
			// for client := range h.clients {
			// 	fmt.Println(client.name)
			// }
			// fmt.Println("-----------------------")
		case message := <-h.broadcast:
			// joined :=[]byte("joined room")
			for client := range h.clients {
				// select {
				// 	case client.send <- message:
				// 	default:
				// 		close(client.send)
				// 		delete(h.clients, client)
				// }
				client.conn.WriteMessage(1, []byte(message))
			}
		case x := <-h.write:
			h.Sendmsg(x)

		}
	}
}

type message struct {
	To   string
	From string
	Msg  string
}

func (c *Client) read() {

	defer func() {
		c.hub.unregister <- c
		c.hub.broadcast <- []byte(c.name)
		// c.conn.Close()
	}()

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		c.hub.write <- msg
		// c.conn.WriteMessage(mType, msg)
	}

}

func (h *Hub) Sendmsg(a []byte) {
	var m message
	er := json.Unmarshal(a, &m)
	if er != nil {
		fmt.Println("error:", er)
	}
	for client := range h.clients {
		if client.name == m.To {
			client.conn.WriteMessage(1, a)
		}
	}
}

func (h *Hub) Base(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func (h *Hub) Socket(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
	// CheckOrigin: func(r *http.Request) bool { return true },				// to access from other sources
	}
	var conn, _ = upgrader.Upgrade(w, r, nil)
	params := mux.Vars(r)
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 256), name: params["user"]}
	go client.read()
	h.register <- client
	h.broadcast <- []byte(client.name)

}

func main() {

	hub := newHub()
	go hub.run()
	r := mux.NewRouter()
	r.HandleFunc("/", hub.Base)
	r.HandleFunc("/ws/{user}", hub.Socket)
	http.Handle("/", r)
	http.ListenAndServe(":3000", nil)

}
