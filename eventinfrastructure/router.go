package eventinfrastructure

import (
	"errors"
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

	//	r.address = GetIP() + ":" + port
	r.address = "localhost:" + port

	return &r
}

func (r *Router) HandleRequest(context echo.Context) error {
	var cr ConnectionRequest
	context.Bind(&cr)

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
		return errors.New("request is missing an address to subscribe to")
	}

	// respond to Subscriber Endpoint
	if len(cr.SubscriberEndpoint) > 0 && len(r.address) > 0 {
		var response ConnectionRequest
		response.PublisherAddr = r.address

		SendConnectionRequest(cr.SubscriberEndpoint, response, true)
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
