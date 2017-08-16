package eventinfrastructure

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/labstack/echo"
	"github.com/xuther/go-message-router/router"
)

type Router struct {
	router              router.Router
	NewSubscriptionChan chan string
	address             string
}

type ConnectionRequest struct {
	PublisherAddr      string `json:"publisher-address"`   // subscribe to my publisher at this address
	SubscriberEndpoint string `json:"subscriber-endpoint"` // hit this endpoint with your publisher address, and I will subscribe to you
}

func NewRouter(routingTable map[string][]string, wg sync.WaitGroup, port string, addrs []string) *Router {
	color.Set(color.FgCyan)
	defer color.Unset()

	var r Router

	r.router = router.Router{}
	err := r.router.Start(routingTable, wg, 1000, []string{}, 120, time.Second*3, port)
	if err != nil {
		color.Set(color.FgHiRed)
		log.Fatalf(err.Error())
	}

	r.NewSubscriptionChan = make(chan string, 5)
	go r.addSubscriptions()

	// subscribe to each of the requested addresses
	for _, addr := range addrs {
		r.NewSubscriptionChan <- addr
	}

	r.address = GetIP() + ":" + port

	return &r
}

func (r *Router) HandleRequest(context echo.Context) error {
	color.Set(color.FgCyan)
	defer color.Unset()

	var cr ConnectionRequest
	context.Bind(&cr)

	err := r.HandleConnectionRequest(cr)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, err.Error())
	}

	return context.JSON(http.StatusOK, nil)
}

func (r *Router) HandleConnectionRequest(cr ConnectionRequest) error {
	color.Set(color.FgGreen)
	defer color.Unset()

	if len(cr.PublisherAddr) > 0 {
		r.NewSubscriptionChan <- cr.PublisherAddr
	} else {
		color.Set(color.FgHiRed)
		log.Printf("[error] request is missing an address to subscribe to")
		color.Unset()
		return errors.New("request is missing an address to subscribe to")
	}

	// respond to Subscriber Endpoint
	if len(cr.SubscriberEndpoint) > 0 && len(r.address) > 0 {
		var response ConnectionRequest
		response.PublisherAddr = r.address
		log.Printf("Sending request to %s", response.PublisherAddr)

		SendConnectionRequest(cr.SubscriberEndpoint, response, true)
	}
	return nil
}

func (r *Router) addSubscriptions() {
	for {
		select {
		case request, ok := <-r.NewSubscriptionChan:
			if !ok {
				color.Set(color.FgHiRed)
				log.Printf("[error] New subscription channel closed")
				color.Unset()
			}
			color.Set(color.FgCyan)
			log.Printf("[router] Adding subscription to %s", request)
			color.Unset()
			r.router.Subscribe(request, 100, 5*time.Second)
		}
	}
}
