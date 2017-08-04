package eventinfrastructure

import (
	"log"
	"net/http"

	"github.com/fatih/color"
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
	color.Set(color.FgBlue)
	defer color.Unset()

	var s Subscriber
	var err error

	s.subscriber, err = subscriber.NewSubscriber(20)
	if err != nil {
		color.Set(color.FgHiRed)
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

func HandleSubscriptionRequest(context echo.Context) error {
	var cr ConnectionRequest
	context.Bind(&cr)

	s := context.Get(ContextSubscriber)
	if sub, ok := s.(*Subscriber); ok {
		if len(cr.PublisherAddr) > 0 {
			sub.subscriptions <- cr.PublisherAddr
		} else {
			return context.JSON(http.StatusBadRequest, "publisher-address can not be empty.")
		}
		return context.JSON(http.StatusOK, nil)
	} else {
		return context.JSON(http.StatusInternalServerError, "subscriber not passed into context.")
	}
}

func (s *Subscriber) addAddresses() {
	for {
		select {
		case addr, ok := <-s.subscriptions:
			color.Set(color.FgBlue)
			if !ok {
				color.Set(color.FgHiRed)
				log.Printf("[error] subscriber address channel closed")
				color.Unset()
			}
			log.Printf("[subscriber] Subscribing to %s", addr)
			s.subscriber.Subscribe(addr, s.filters)
			color.Unset()
		}
	}
}

func (s *Subscriber) read() {
	for {
		message := s.subscriber.Read()

		color.Set(color.FgBlue)
		log.Printf("[subscriber] Recieved message: %s", message)
		color.Unset()

		s.MessageChan <- message
	}
}
