package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
)

func NewHub() *Hub {
	hub := &Hub{
		userCnt:    atomic.Int32{},
		clients:    make(map[*Client]bool),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
		broadcast:  make(chan []byte, 16),
		close:      make(chan struct{}),
	}

	go hub.Run()

	return hub
}

func (h *Hub) Run() {
	log.Printf("Chatroom hub hosting.......")
	for {
		select {
		case client := <-h.register:
			var num int = int(h.userCnt.Load())
			s := fmt.Sprintf("GUEST %d", num)
			log.Printf("%s join...", s)
			client.name = s
			client.hub = h
			h.clients[client] = true
			h.userCnt.Add(1)
			log.Printf("a new client connected, current user count: %d\n", h.userCnt.Load())

			payload, err := json.Marshal(Payload{
				User:    "system",
				Message: "name:" + s,
			})

			if err != nil {
				log.Println(err)
				continue
			}

			client.send <- payload

			info := fmt.Sprintf("GUEST %d join the channel", num)
			payload, err = json.Marshal(Payload{
				User:    "system",
				Message: info,
			})

			h.broadcast <- payload
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				s := client.name
				err := client.conn.Close()
				if err != nil {
					log.Printf("error closing client connection: %v\n", err)
				}

				delete(h.clients, client)
				h.userCnt.Add(-1)
				log.Printf("a client disconnected, current user count: %d\n", h.userCnt.Load())

				info := fmt.Sprintf("%s leave the channel", s)
				payload, err := json.Marshal(Payload{
					User:    "system",
					Message: info,
				})

				h.broadcast <- payload
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				client.send <- message
			}
		case <-h.close:
			for client := range h.clients {
				client.conn.Close()
			}

			log.Printf("Hub shutdown...")
			return
		}
	}
}

type Hub struct {
	userCnt    atomic.Int32
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	close      chan struct{}
}
