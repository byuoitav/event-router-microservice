package subscription

import (
	"log"

	"github.com/byuoitav/go-message-router/router"
)

var R router.Router

type SubscribeRequest struct {
	Address    string `json:"address"`
	PubAddress string `json:"paddress,omitempty"`
}

func Subscribe(sr SubscribeRequest) error {
	log.Printf("Subscription Request from %s", sr.Address)
	R.Subscribe(sr.Address, 5, 3)
	log.Printf("Subscribed to %s!", sr.Address)

	if len(sr.PubAddress) != 0 {
		log.Printf("subscribing to %s", sr.PubAddress)
		log.Printf("not implemented yet")
	}
	return nil
}
