package eventinfrastructure

import (
	"log"
	"net/http"

	"github.com/labstack/echo"
	"github.com/xuther/go-message-router/common"
	"github.com/xuther/go-message-router/subscriber"
)

type Subscriber struct {
	subscriber    subscriber.Subscriber
	filters       []string
	subscriptions chan string
	MessageChan   chan common.Message
}

func NewSubscriber(filters []string, requests ...string) *Subscriber {
	var s Subscriber
	var err error

	s.subscriber, err = subscriber.NewSubscriber(20)
	if err != nil {
		log.Fatalf("[error] Failed to create subscriber. error: %s", err.Error())
	}

	s.filters = filters
	s.subscriptions = make(chan string, 10)
	go s.addAddresses()

	// subscribe to each of the requested addresses
	for _, addr := range requests {
		s.subscriptions <- addr
	}

	// read messages
	s.MessageChan = make(chan common.Message, 20)
	go s.read()

	return &s
}

func (s *Subscriber) HandleSubscriptionRequest(context echo.Context) error {
	var cr ConnectionRequest
	context.Bind(&cr)

	if len(cr.PublisherAddr) > 0 {
		s.subscriptions <- cr.PublisherAddr
	} else {
		return context.JSON(http.StatusBadRequest, "publisher-address can not be empty.")
	}
	return context.JSON(http.StatusOK, nil)
}

func (s *Subscriber) addAddresses() {
	for {
		select {
		case addr, ok := <-s.subscriptions:
			if !ok {
				log.Printf("[error] subscriber address channel closed")
			}
			log.Printf("[subscriber] Subscribing to %s", addr)
			s.subscriber.Subscribe(addr, s.filters)
		}
	}
}

func (s *Subscriber) read() {
	for {
		message := s.subscriber.Read()
		log.Printf("[subscriber] Recieved message: %s", message)
		s.MessageChan <- message
	}
}
