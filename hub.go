package main

import (
	"context"
	"log"
	"time"
)

type Hub struct {
	clients    map[string]*UserClient
	register   chan *UserClient
	unregister chan *UserClient
	database   *Database
}

func newHub(db *Database) *Hub {
	return &Hub{
		clients:    make(map[string]*UserClient),
		register:   make(chan *UserClient),
		unregister: make(chan *UserClient),
		database:   db,
	}
}

func (h *Hub) run() {
	go func() {
		for {
			select {
			case client := <-h.register:
				// TODO use session key rather than `client.id`
				h.clients[client.id] = client
				// client.signInComplete <- true
				h.database.userTable[client.id] = &UserData{
					Token: client.token,
				}

				log.Printf("Client %d registered", client.id)

			case client := <-h.unregister:
				if _, ok := h.clients[client.id]; ok {
					delete(h.clients, client.id)
				}
			}
		}
	}()
	go h.syncState()
}

func (h *Hub) syncState() {
	for {
		for _, client := range h.clients {
			curr, err := client.spotifyClient.PlayerCurrentlyPlaying(context.Background())
			if err != nil {
				log.Println(err)
			}
			if curr.Playing {
				ts := curr.Progress / 1000
				log.Printf("Song: %s [%02d:%02d]", curr.Item.Name, ts/60, ts%60)
			}

			time.Sleep(5 * time.Second)

		}
	}
}
