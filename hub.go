package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
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
			h.clients[client] = true
			h.userCnt.Add(1)
			log.Printf("a new client connected, current user count: %d\n", h.userCnt.Load())

			var num int = int(h.userCnt.Load())
			s := fmt.Sprintf("%s%d", GetRandomName(0), num)
			log.Printf("%s join...", s)
			client.name = s
			client.hub = h

			payload, err := json.Marshal(Payload{
				User:    "system_name",
				Message: client.name,
			})

			if err != nil {
				log.Println(err)
				continue
			}

			client.send <- payload

			payload, err = json.Marshal(Payload{
				User:    "system_join",
				Message: client.name,
			})

			h.broadcast <- payload

			payload, err = json.Marshal(Payload{
				User:    "system_count",
				Message: strconv.Itoa(num),
			})

			h.broadcast <- payload
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				err := client.conn.Close()
				if err != nil {
					log.Printf("error closing client connection: %v\n", err)
				}

				delete(h.clients, client)
				h.userCnt.Add(-1)
				log.Printf("%s disconnected, current user count: %d\n", client.name, h.userCnt.Load())
				num := int(h.userCnt.Load())

				payload, err := json.Marshal(Payload{
					User:    "system_leave",
					Message: client.name,
				})

				h.broadcast <- payload

				payload, err = json.Marshal(Payload{
					User:    "system_count",
					Message: strconv.Itoa(num),
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
