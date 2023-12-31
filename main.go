package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/PoteeDev/admin/api/database"
	"github.com/PoteeDev/events-stream/models"
	"github.com/PoteeDev/events-stream/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RoundInfo struct {
	TeamName string              `json:"team_name,omitempty"`
	TeamHost string              `json:"team_host,omitempty"`
	Services map[string]Services `json:"services,omitempty"`
}

type Services struct {
	PingStatus int                `json:"ping_status,omitempty"`
	Checkers   map[string]Checker `json:"checkers,omitempty"`
	Exploits   map[string]Exploit `json:"exploits,omitempty"` //exploit name and status
}

type Checker struct {
	GetStatus int `json:"get_status,omitempty"`
	PutStatus int `json:"put_status,omitempty"`
}

type Exploit struct {
	Cost   int `json:"cost,omitempty"`
	Status int `json:"status,omitempty"`
}

type Events struct {
	Events map[string]RoundInfo `bson:"events" json:"events"`
}

type documentKey struct {
	ID primitive.ObjectID `bson:"_id"`
}

type changeID struct {
	Data string `bson:"_data"`
}

type namespace struct {
	Db   string `bson:"db"`
	Coll string `bson:"coll"`
}

type changeEvent struct {
	ID            changeID            `bson:"_id"`
	OperationType string              `bson:"operationType"`
	ClusterTime   primitive.Timestamp `bson:"clusterTime"`
	FullDocument  Events              `bson:"fullDocument"`
	DocumentKey   documentKey         `bson:"documentKey"`
	Ns            namespace           `bson:"ns"`
}

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
		var changeEvent changeEvent

		err := cs.Decode(&changeEvent)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(changeEvent)
		pool.Broadcast <- websocket.Message{Message: ConvertTeamsEvents(changeEvent)}
	}
}

func ConvertTeamsEvents(events changeEvent) *models.TeamsEvents {
	teamsEvents := models.TeamsEvents{}
	for _, event := range events.FullDocument.Events {
		services := []models.TeamServices{}
		for serviceName, service := range event.Services {
			checkers := []models.Checkers{}
			for _, checker := range service.Checkers {
				checkers = append(checkers, models.Checkers{
					PutStatus: checker.PutStatus,
					GetStatus: checker.GetStatus,
				})
			}
			exploits := []models.Exploits{}
			for _, exploit := range service.Exploits {
				if exploit.Status >= 0 {
					exploits = append(exploits, models.Exploits{
						Status: exploit.Status,
					})
				}
			}
			services = append(services, models.TeamServices{
				Name:     serviceName,
				Checkers: checkers,
				Exploits: exploits,
			})
		}

		teamsEvents.Teams = append(teamsEvents.Teams, models.Teams{
			Name:     event.TeamName,
			Services: services,
		})
	}
	return &teamsEvents
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
