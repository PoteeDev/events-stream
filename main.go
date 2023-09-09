package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/PoteeDev/admin/api/database"
	"github.com/PoteeDev/events-stream/websocket"
	"go.mongodb.org/mongo-driver/mongo"
)

var ctx = context.Background()

func watchEvents(pool *websocket.Pool) {
	client := database.ConnectDB()
	coll := database.GetCollection(client, "events")
	cs, err := coll.Watch(ctx, mongo.Pipeline{})
	if err != nil {
		log.Fatalln(err)
	}
	defer cs.Close(ctx)
	for cs.Next(context.TODO()) {
		fmt.Println(cs.Current)
		pool.Broadcast <- websocket.Message{Message: string(cs.Current)}
	}
}

func serveWs(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {
	fmt.Println("WebSocket Endpoint Hit")
	conn, err := websocket.Upgrade(w, r)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	client := &websocket.Client{
		Conn: conn,
		Pool: pool,
	}

	pool.Register <- client
	client.Read()
}

func setupRoutes() {
	pool := websocket.NewPool()
	go pool.Start()
	go watchEvents(pool)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(pool, w, r)
	})
}

func main() {
	fmt.Println("server started")
	setupRoutes()
	http.ListenAndServe(":8080", nil)
}
