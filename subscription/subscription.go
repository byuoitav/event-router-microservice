package subscription

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/xuther/go-message-router/router"
)

var R router.Router
var Hostname string

type SubscribeRequest struct {
	Address    string `json:"subtomeat"`
	PubAddress string `json:"sendbackto,omitempty"`
}

func Subscribe(sr SubscribeRequest) error {
	log.Printf("[new] Subscribing to %s", sr.Address)
	R.Subscribe(sr.Address, 5, 3)

	if len(sr.PubAddress) != 0 {
		log.Printf("Telling %s to subscribe to me", sr.PubAddress)

		var s SubscribeRequest
		s.Address = Hostname + ".byu.edu:7000"
		body, err := json.Marshal(s)
		if err != nil {
			return err
		}

		_, err = http.Post("http://"+sr.PubAddress, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return err
		}
	}
	return nil
}
