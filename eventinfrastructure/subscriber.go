package eventinfrastructure

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/xuther/go-message-router/subscriber"
)

type Subscriber struct {
	subscriber          subscriber.Subscriber
	newSubscriptionChan chan SubscriptionRequest
}

type SubscriptionRequest struct {
	address string
	filters []string
}

type ConnectionRequest struct {
	PublisherAddr      string `json:"publisher-address"`   // subscribe to my publisher at this address
	SubscriberEndpoint string `json:"subscriber-endpoint"` // hit this endpoint with your publisher address, and I will subscribe to you
	// omitempty
}

/* sooo....
handlers for each microservice should turn a connection request into a subscription request, and then add that request into their NewSubscriptionChan.

After doing that, if there is a Subscriber Endpoint attached, the handler should respond by sending their publishers address to that endpoint.
*/

// an endpoint for others to subscribe to it's publisher, if one exists
func (s Subscriber) Start(requests ...SubscriptionRequest) {
	sub, err := subscriber.NewSubscriber(20)
	if err != nil {
		log.Fatalf("[error] Failed to create subscriber. error: %s", err.Error())
	}

	s.subscriber = sub
	s.addSubscriptions()

	for _, sr := range requests {
		s.newSubscriptionChan <- sr
	}

	for {
		message := s.subscriber.Read()
		log.Printf("[subscriber] Recieved message: %s", message)
		// write it back into a diffrent channel for someone else to read from?
	}
}

func (s Subscriber) HandleConnectionRequest(cr ConnectionRequest, filters []string, publisherAddr string) error {
	var sr SubscriptionRequest
	sr.address = cr.PublisherAddr
	sr.filters = filters
	s.newSubscriptionChan <- sr

	// respond to Subscriber Endpoint
	if len(cr.SubscriberEndpoint) > 0 && len(publisherAddr) > 0 {
		var response ConnectionRequest
		response.PublisherAddr = publisherAddr

		body, err := json.Marshal(response)
		if err != nil {
			return err
		}

		res, err := http.Post(cr.SubscriberEndpoint, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		if res.StatusCode != 200 {
			log.Printf("[error] response from %s: %v", cr.SubscriberEndpoint, res)
		}
	}
	return nil
}

/*
func HandleConnectionRequest(context echo.Context) error {
	var cr ConnectionRequest
	context.Bind(&cr)

	return context.JSON(http.StatusOK, "success")
}
*/

func (s Subscriber) addSubscriptions() {
	s.newSubscriptionChan = make(chan SubscriptionRequest, 5)

	go func() {
		for {
			select {
			case request, ok := <-s.newSubscriptionChan:
				if !ok {
					log.Printf("[error] New subscription channel closed")
				}
				// handle request for new subscription
				log.Printf("[subscriber] Starting subscription to %s", request.address)
				err := s.subscriber.Subscribe(request.address, request.filters)
				for err != nil {
					log.Printf("[error] failed to subscribe. Error %s", err)
					log.Printf("trying again in 5 seconds...")
					time.Sleep(5 * time.Second)

					err = s.subscriber.Subscribe(request.address, request.filters)
				}
			}
		}
	}()
}
