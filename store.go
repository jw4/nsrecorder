package nsrecorder // import "jw4.us/nsrecorder"

import (
	"log"
	"time"
)

type Store interface {
	Accept([]Client, []Lookup) error
}

type Client struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type Lookup struct {
	When    time.Time `json:"when"`
	Client  string    `json:"client"`
	Host    string    `json:"host"`
	Type    string    `json:"type"`
	FirstIP string    `json:"first_ip"`
}

func NewLogStore() Store {
	return &logStore{}
}

type logStore struct{}

func (*logStore) Accept(clients []Client, lookups []Lookup) error {
	log.Printf("ACCEPT %d clients, %d lookups", len(clients), len(lookups))
	for _, client := range clients {
		log.Printf("\tclient: %+v", client)
	}
	for _, lookup := range lookups {
		log.Printf("\tlookup: %+v", lookup)
	}
	return nil
}
