package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Client struct {
	hub *Hub

	conn *websocket.Conn

	send chan []byte //buffered chan

	// name string

	id int

	status bool
}

type Hub struct {
	clients map[int]*Client // registered clients

	broadcast chan []byte

	register chan int

	unregister chan int

	write chan []byte
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan int),
		unregister: make(chan int),
		clients:    make(map[int]*Client),
		write:      make(chan []byte),
	}
}

func (h *Hub) run() {
	for {
		select {
		case clientid := <-h.register:
			h.clients[clientid].status = true
			// fmt.Println("---------ACTIVE--------------")
			// for client := range h.clients {
			// 	fmt.Println(h.clients[client].id)
			// }
			// fmt.Println("-----------------------")
		case clientid := <-h.unregister:
			// 	fmt.Println("second")
			if _, ok := h.clients[clientid]; ok {
				delete(h.clients, clientid)
				// h.clients[client].conn.Close()
				// close(client.send)
			}
			// fmt.Println("---------ACTIVE--------------")
			// for client := range h.clients {
			// 	fmt.Println(h.clients[client].id)
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
				h.clients[client].conn.WriteMessage(1, []byte(message))
			}
			// case x := <-h.write:
			// h.Sendmsg(x)

		}
	}
}

type message struct {
	To    string
	From  string
	Image string
	Msg   string
}

func (c *Client) read() {

	defer func() {
		c.hub.unregister <- c.id
		c.hub.broadcast <- []byte(strconv.Itoa(c.id))
		c.conn.Close()
	}()

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var m message
		er := json.Unmarshal(msg, &m)
		if er != nil {
			fmt.Println("error:", er)
		}
		to, err := strconv.Atoi(m.To)
		if err != nil {
			return
		}

		fmt.Println(m.Msg)
		fmt.Println(m.Image)

		c.hub.send(to, msg)

		// c.hub.send(to, []byte(m.Msg))		//only msg

		// c.hub.client[m.To].send <- msg

		// c.hub.write <- msg
		// c.conn.WriteMessage(1, msg)
	}

}

func (h *Hub) send(id int, msg []byte) {

	h.clients[id].conn.WriteMessage(1, msg)

	// for{
	// 	select{
	// 		case m := <- c.send:
	// 		fmt.Println("recieved")
	// 	}
	// }

}

func (h *Hub) Base(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "demo.html")
}

func (h *Hub) Socket(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
	// CheckOrigin: func(r *http.Request) bool { return true },				// to access from other sources
	}
	var conn, _ = upgrader.Upgrade(w, r, nil)
	params := mux.Vars(r)
	i, err := strconv.Atoi(params["id"])
	if err != nil {
		return
	}
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 256), id: i, status: true}
	// fmt.Println(client)
	h.clients[i] = client
	go client.read()
	h.register <- client.id
	h.broadcast <- []byte(strconv.Itoa(client.id))

}

func main() {

	hub := newHub()
	go hub.run()
	r := mux.NewRouter()
	r.HandleFunc("/", hub.Base)
	r.HandleFunc("/ws/{id:[0-9]+}", hub.Socket)
	http.Handle("/", r)
	http.ListenAndServe(":3000", nil)
}
