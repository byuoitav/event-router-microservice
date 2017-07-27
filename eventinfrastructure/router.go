package eventinfrastructure

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/xuther/go-message-router/router"
)

type Router struct {
	router              router.Router
	newSubscriptionChan chan string
	address             string
}

type ConnectionRequest struct {
	PublisherAddr      string `json:"publisher-address"`   // subscribe to my publisher at this address
	SubscriberEndpoint string `json:"subscriber-endpoint"` // hit this endpoint with your publisher address, and I will subscribe to you
}

func NewRouter(routingTable map[string][]string, wg sync.WaitGroup, port string, addrs ...string) *Router {
	var r Router

	r.router = router.Router{}
	err := r.router.Start(routingTable, wg, 1000, []string{}, 120, time.Second*3, port)
	if err != nil {
		log.Fatalf(err.Error())
	}

	r.newSubscriptionChan = make(chan string, 5)
	go r.addSubscriptions()

	// subscribe to each of the requested addresses
	for _, addr := range addrs {
		r.newSubscriptionChan <- addr
	}

	r.address = GetIP() + ":" + port

	return &r
}

func (r *Router) HandleRequest(context echo.Context) error {
	var cr ConnectionRequest
	context.Bind(&cr)
	log.Printf("[router] Recieved subscription request for %s", cr.PublisherAddr)

	err := r.HandleConnectionRequest(cr)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, err.Error())
	}

	return context.JSON(http.StatusOK, nil)
}

func (r *Router) HandleConnectionRequest(cr ConnectionRequest) error {
	if len(cr.PublisherAddr) > 0 {
		r.newSubscriptionChan <- cr.PublisherAddr
	} else {
		log.Printf("[error] request is missing an address to subscribe to")
	}

	// respond to Subscriber Endpoint
	if len(cr.SubscriberEndpoint) > 0 && len(r.address) > 0 {
		var response SubscriptionRequest
		response.Address = r.address

		body, err := json.Marshal(response)
		if err != nil {
			return err
		}

		resp, err := http.Post(cr.SubscriberEndpoint, "application/json", bytes.NewBuffer(body))
		for err != nil || resp.StatusCode != 200 {
			if resp != nil {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Printf("[error] failed to post. Response: (%v) %s", resp.StatusCode, body)
			} else {
				log.Printf("[error] failed to post. Error: %s", err.Error())
			}

			log.Printf("Trying again in 5 seconds.")
			time.Sleep(5 * time.Second)
			resp, err = http.Post(cr.SubscriberEndpoint, "application/json", bytes.NewBuffer(body))
		}

		log.Printf("[router] Successfully posted connection request to %s", cr.SubscriberEndpoint)
		resp.Body.Close()
	}
	return nil
}

func (r *Router) addSubscriptions() {
	for {
		select {
		case request, ok := <-r.newSubscriptionChan:
			if !ok {
				log.Printf("[error] New subscription channel closed")
			}
			log.Printf("[router] Adding subscription to %s", request)
			r.router.Subscribe(request, 10, 3)
		}
	}
}
