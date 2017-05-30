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
	Address    string `json:"subtome"`
	PubAddress string `json:"sendbackto,omitempty"`
}

func Subscribe(sr SubscribeRequest) error {
	log.Printf("[new] Subscribing to %s", sr.Address)
	R.Subscribe(sr.Address, 5, 3)

	if len(sr.PubAddress) != 0 {
		var s SubscribeRequest
		s.Address = Hostname + ".byu.edu:7000"
		body, err := json.Marshal(s)
		if err != nil {
			return err
		}

		log.Printf("[response] Telling %s to subscribe to me, with body %s", sr.PubAddress, s.Address)
		resp, err := http.Post("http://"+sr.PubAddress, "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("[response][error] Failed to send post request: %s", err.Error())
			return err
		}
		if resp != nil {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				log.Printf("[response] Post to %s successful", sr.PubAddress)
			} else {
				log.Printf("[response][error] post to %s unsuccessful. Response:\n %s", sr.PubAddress, resp)
			}
		}
	}
	return nil
}
