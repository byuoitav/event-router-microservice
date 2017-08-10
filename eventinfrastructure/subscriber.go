package eventinfrastructure

import (
	"errors"
	"log"

	"github.com/fatih/color"
	"github.com/xuther/go-message-router/common"
	"github.com/xuther/go-message-router/publisher"
	"github.com/xuther/go-message-router/subscriber"
)

type Subscriber struct {
	subscriber    subscriber.Subscriber
	filters       []string
	subscriptions chan string
	MessageChan   chan common.Message

	/*
		if a "test" event is recieved, then
		an appropriate response will be sent to the router
		through this publisher.

		if it is null, then a webhook (pointing to
		the device-monitoring-microserivce) will be hit
		with the same information.
	*/
	pub *publisher.Publisher
}

func NewSubscriber(filters []string, pub *publisher.Publisher, requests ...string) *Subscriber {
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
	s.pub = pub
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

func HandleSubscriptionRequest(cr ConnectionRequest, sub *Subscriber) error {
	if len(cr.PublisherAddr) > 0 {
		sub.subscriptions <- cr.PublisherAddr
	} else {
		return errors.New("publisher-address can not be empty.")
	}

	return nil
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

		header := string(message.MessageHeader[:])
		log.Printf("message header %s", header)
		if header == eventType.TEST {

		}

		color.Set(color.FgBlue)
		log.Printf("[subscriber] Recieved message: %s", message)
		color.Unset()

		s.MessageChan <- message
	}
}
