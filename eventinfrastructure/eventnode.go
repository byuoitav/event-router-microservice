package eventinfrastructure

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/byuoitav/device-monitoring-microservice/statusinfrastructure"
	"github.com/fatih/color"
	"github.com/xuther/go-message-router/common"
	"github.com/xuther/go-message-router/publisher"
	"github.com/xuther/go-message-router/subscriber"
)

type EventNode struct {
	Name  string
	Port  string
	Read  chan common.Message
	Write chan common.Message

	subscriber    subscriber.Subscriber
	filters       []string
	subscriptions chan string

	publisher publisher.Publisher
}

// filters: an array of strings to filter events recieved by
// port: a unique port to publish events on
// addrs: addresses of subscriber to subscribe to
// name: name of event node
func NewEventNode(name, port string, filters []string, addrs ...string) *EventNode {
	color.Set(color.FgBlue)
	defer color.Unset()
	var n EventNode

	n.Port = port
	n.Name = name

	//
	// create subscriber
	var err error

	n.subscriber, err = subscriber.NewSubscriber(20)
	if err != nil {
		color.Set(color.FgHiRed)
		log.Fatalf("Failed to create subscriber. error: %s", err.Error())
	}

	// add respose filter to all microservices
	addFilter := true
	for _, filter := range filters {
		if strings.EqualFold(filter, TestPleaseReply) {
			addFilter = false
		}
	}
	if addFilter {
		filters = append(filters, TestPleaseReply)
	}
	n.filters = filters

	n.subscriptions = make(chan string, 10)
	go n.addSubscriptions()

	// subscribe to each of the requested addresses
	for _, addr := range addrs {
		n.subscriptions <- addr
	}

	// read messages
	n.Read = make(chan common.Message, 20)
	go n.read()

	//
	// create publisher
	n.publisher, err = publisher.NewPublisher(n.Port, 100, 10)
	if err != nil {
		color.Set(color.FgHiRed)
		log.Fatalf("[error] Failed to create publisher. error: %s", err.Error())
	}

	n.Write = make(chan common.Message)

	// listen and write
	go n.publisher.Listen()
	go n.write()

	color.Set(color.FgGreen, color.Bold)
	log.Printf("Event node '%s' created. Writing events on port: %s", n.Name, n.Port)
	color.Unset()

	return &n
}

func (n *EventNode) PublishEvent(e Event, eventType string) error {
	toSend, err := json.Marshal(e)
	if err != nil {
		return err
	}

	header := [24]byte{}
	copy(header[:], []byte(eventType))

	n.Write <- common.Message{MessageHeader: header, MessageBody: toSend}
	return nil
}

func (n *EventNode) PublishMessageByEventType(eventType string, body []byte) {
	header := [24]byte{}
	copy(header[:], []byte(eventType))

	n.Write <- common.Message{MessageHeader: header, MessageBody: body}
}

func (n *EventNode) PublishJSONMessageByEventType(eventType string, i interface{}) error {
	header := [24]byte{}
	copy(header[:], []byte(eventType))

	body, err := json.Marshal(i)
	if err != nil {
		return err
	}

	n.Write <- common.Message{MessageHeader: header, MessageBody: body}
	return nil
}

func (n *EventNode) PublishMessage(m common.Message) {
	n.Write <- m
}

func HandleSubscriptionRequest(cr ConnectionRequest, n *EventNode) error {
	if len(cr.PublisherAddr) > 0 {
		color.Set(color.FgYellow, color.Bold)
		log.Printf("Subscribing to %s", cr.PublisherAddr)
		color.Unset()

		n.subscriptions <- cr.PublisherAddr
	} else {
		return errors.New("publisher-address can not be empty.")
	}

	/*
		if len(cr.SubscriberEndpoint) > 0 {
			color.Set(color.FgYellow)
			log.Printf("Responding to %s's subscription request @ %s", cr.PublisherAddr, cr.SubscriberEndpoint)
			color.Unset()
		}
	*/

	return nil
}

func (n *EventNode) addSubscriptions() {
	for {
		select {
		case addr, ok := <-n.subscriptions:
			color.Set(color.FgBlue)
			if !ok {
				color.Set(color.FgHiRed)
				log.Printf("[error] subscriber address channel closed")
				color.Unset()
			}
			log.Printf("[subscriber] Subscribing to %s", addr)
			n.subscriber.Subscribe(addr, n.filters)
			color.Unset()
		}
	}
}

func (n *EventNode) read() {
	for {
		message := n.subscriber.Read()

		header := string(bytes.Trim(message.MessageHeader[:], "\x00"))
		if strings.EqualFold(header, TestPleaseReply) {
			var s statusinfrastructure.EventNodeStatus
			s.Name = n.Name

			n.PublishJSONMessageByEventType(TestReply, s)
		} else {
			color.Set(color.FgBlue)
			log.Printf("[subscriber] Recieved message: %s", message)
			color.Unset()

			n.Read <- message
		}

	}
}

func (n *EventNode) write() {
	for {
		select {
		case message, ok := <-n.Write:
			if !ok {
				color.Set(color.FgHiRed)
				log.Fatalf("[error] publisher write channel closed")
			}

			color.Set(color.FgMagenta)
			log.Printf("Publishing message: %s", message)

			err := n.publisher.Write(message)
			if err != nil {
				color.Set(color.FgHiRed)
				log.Printf("[error] error publishing message: %s", err.Error())
			}
			color.Unset()
		}
	}
}
