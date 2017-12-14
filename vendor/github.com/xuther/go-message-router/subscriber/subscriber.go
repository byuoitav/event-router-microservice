package subscriber

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/gorilla/websocket"

	"github.com/xuther/go-message-router/common"
)

const debug = false

type Subscriber interface {
	Subscribe(address string, filter []string) error
	GetSubscriptions() []string
	Read() common.Message
	Close()
}

type subscriber struct {
	aggregateQueueSize int
	retry              bool
	maxRetryInterval   int
	subscriptions      []*readSubscription
	QueuedMessages     chan common.Message
	subscribeChan      chan *readSubscription
	unsubscribeChan    chan *readSubscription
}

type readSubscription struct {
	FilterList []*regexp.Regexp
	RawFilters []string
	Address    string
	Sub        *subscriber
	Connection *websocket.Conn
}

func NewSubscriber(aggregateQueueSize int) (Subscriber, error) {

	toReturn := subscriber{
		aggregateQueueSize: aggregateQueueSize,
		QueuedMessages:     make(chan common.Message, aggregateQueueSize),
		subscribeChan:      make(chan *readSubscription, 10),
		unsubscribeChan:    make(chan *readSubscription, 10),
		retry:              true,
		maxRetryInterval:   500,
	}
	toReturn.startSubscriptionManager()

	return &toReturn, nil
}

func (s *subscriber) Close() {
}

func (s *subscriber) Read() common.Message {
	return <-s.QueuedMessages
}

func (s *subscriber) GetSubscriptions() []string {
	toReturn := []string{}
	for _, v := range s.subscriptions {
		toReturn = append(toReturn, v.Address)
	}
	return toReturn
}

func (s *subscriber) Subscribe(address string, filters []string) error {

	//we need to make sure it's not already in our subscriptions
	for _, s := range s.subscriptions {
		if debug {
			log.Printf("checking subscription for %v. Comparing to %v", s.Address, address)
		}
		if s.Address == address {
			log.Printf("Subscription already exists for %v", s.Address)
			return nil
		}
	}

	compiledFilters := []*regexp.Regexp{}
	for _, filter := range filters {
		val, err := regexp.Compile(filter)
		if err != nil {
			err = errors.New(fmt.Sprintf("Error compiling filter %s: %s", filter, err.Error()))
			return err
		}
		compiledFilters = append(compiledFilters, val)
	}

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial(fmt.Sprintf("ws://%v/subscribe", address), nil)
	if err != nil {
		return err
	}

	subscription := readSubscription{
		Address:    address,
		Sub:        s,
		Connection: conn,
		FilterList: compiledFilters,
		RawFilters: filters,
	}

	log.Printf("Connection sent")
	log.Printf("Subscription created, adding to manager")
	s.subscribeChan <- &subscription
	return nil
}

//do a decaying retry on the connection, with a timeout (in seconds) by set in the subscribe struct
func (s *subscriber) retryConnection(sub *readSubscription) {
	//start at once a second, then double it until we hit the timeout limit
	timer := 1
	for {
		log.Printf("will attempt reconnect in %v seconds", timer)
		time.Sleep(time.Duration(timer) * time.Second)
		log.Printf("Attempting reconnect with %v", sub.Address)

		//try the reconnect
		err := s.Subscribe(sub.Address, sub.RawFilters)
		if err != nil {
			log.Printf("Could not reestablish connection: %v", err.Error())

			if timer < s.maxRetryInterval {
				temptimer := (float64(timer) * 1.5) + .5
				log.Printf("temp timer: %v", temptimer)
				timer = int(temptimer)

				continue
			}
		}

		//it was successful
		log.Printf("Connection reestablished with %v", sub.Address)
		return
	}
}

func (s *subscriber) startSubscriptionManager() {
	log.Printf("Starting subscription Manager.")
	go func() {
		for {
			select {
			case sub := <-s.subscribeChan:
				log.Printf("Subscribed to %v.", sub.Address)
				s.subscriptions = append(s.subscriptions, sub)
				sub.StartListener()

			case unsub := <-s.unsubscribeChan:
				for k, i := range s.subscriptions {
					if i == unsub { //remove from our list of subscribers
						s.subscriptions[k] = s.subscriptions[len(s.subscriptions)-1]
						s.subscriptions = s.subscriptions[:len(s.subscriptions)-1]
						if s.retry {
							log.Printf("adding connection to Retry queue")
							go s.retryConnection(i)
						}
					}
				}
			}
		}
	}()
}

func (rs *readSubscription) StartListener() {
	go func() {
		log.Printf("Starting listener")
		for {

			var message common.Message
			err := rs.Connection.ReadJSON(&message)
			if err != nil {
				if err, ok := err.(*websocket.CloseError); ok {
					log.Printf("The connection for %s was closed. %v", rs.Address, err.Error())
					rs.Sub.unsubscribeChan <- rs
					return
				} else {
					log.Printf("There was some problem with the connection to %v. Will attempt reconnect", rs.Address)
					rs.Sub.unsubscribeChan <- rs
					return
				}
			}

			//check the header to see if we care about the message
			for _, filter := range rs.FilterList {

				//only read in the message if it meets the criteria
				if filter.Match(message.MessageHeader[:]) {
					rs.Sub.QueuedMessages <- message
					break
				}
			}
		}
	}()
}
