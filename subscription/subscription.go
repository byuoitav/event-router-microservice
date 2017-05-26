package subscription

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/xuther/go-message-router/router"
)

var R router.Router

type SubscribeRequest struct {
	Address    string `json:"subscribeto"`
	PubAddress string `json:"subscribetome,omitempty"`
}

func Subscribe(sr SubscribeRequest) error {
	log.Printf("[new] Subscribing to %s", sr.Address)
	R.Subscribe(sr.Address, 5, 3)

	if len(sr.PubAddress) != 0 {
		log.Printf("Telling %s to subscribe to me", sr.PubAddress)

		var s SubscribeRequest
		s.Address = sr.PubAddress
		body, err := json.Marshal(s)
		if err != nil {
			return err
		}

		resp, err := http.Post(sr.PubAddress+"/subscribe", "application/json", bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		log.Printf("[debug] response: %s", resp)
	}
	return nil
}
