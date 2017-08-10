package eventinfrastructure

import (
	"bytes"
	"errors"
	"log"
	"strings"

	"github.com/fatih/color"
	"github.com/xuther/go-message-router/common"
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
	pub *Publisher
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

	// add Test to all subscribers
	filters = append(filters, Test)
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

func (s *Subscriber) TiePublisher(p *Publisher) {
	s.pub = p
}

func (s *Subscriber) GetPublisher() *Publisher {
	return s.pub
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

		header := string(bytes.Trim(message.MessageHeader[:], "\x00"))
		if strings.EqualFold(header, Test) {
			// other stuff with it, don't put it into the message channel
			// works to here:)
		} else {
			color.Set(color.FgBlue)
			log.Printf("[subscriber] Recieved message: %s", message)
			color.Unset()

			s.MessageChan <- message
		}

	}
}
