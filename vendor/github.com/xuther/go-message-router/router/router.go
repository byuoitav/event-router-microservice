package router

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/xuther/go-message-router/common"
	"github.com/xuther/go-message-router/publisher"
	"github.com/xuther/go-message-router/subscriber"
)

type Router struct {
	addressChan   chan subscriptionReq
	inChan        chan common.Message
	outChan       chan common.Message
	exitChan      chan bool
	subscriptions []string
}

type subscriptionReq struct {
	Address  string
	Count    int
	Interval time.Duration
}

var subscriptionHeader = [24]byte{'X',
	'S', 'U', 'B', 'S',
	'C', 'R', 'I', 'P',
	'T', 'I', 'O', 'N',
	'R', 'E', 'Q',
}

const debug = false

var subscriptionHeaderString = "XSUBSCRIPTIONREQ"

func (r *Router) publisher(port string, wg sync.WaitGroup) error {

	log.Printf("Publisher: Starting publisher")
	pub, err := publisher.NewPublisher(port, 1000, 10)
	if err != nil {
		wg.Done()
		return err
	}

	log.Printf("Publisher: Publisher ready.")
	go func() {
		for {
			select {
			case message := <-r.outChan:
				pub.Write(message)
			case <-r.exitChan:
				wg.Done()
				return
			}
		}
	}()
	go pub.Listen()
	return nil
}

func (r *Router) reciever(messageTypes []string, wg sync.WaitGroup) error {
	log.Printf("Reciever: staring reciever")

	sub, err := subscriber.NewSubscriber(3000)
	if err != nil {
		wg.Done()
		return err
	}

	go func() {
		messageTypes = append(messageTypes, subscriptionHeaderString)
		for {
			select {
			case addr, ok := <-r.addressChan:
				if ok {
					alreadySubbed := false
					log.Printf("Reciever: Starting connection with %s", addr.Address)
					for _, t := range sub.GetSubscriptions() {
						log.Printf("Checking already existent subscription")
						if t == addr.Address {
							alreadySubbed = true
							break
						} else {
						}
					}
					if alreadySubbed {
						log.Printf("Receiver: already subbed to %s", addr.Address)
						continue
					}

					err = sub.Subscribe(addr.Address, messageTypes)
					if err != nil {
						if addr.Count > 0 { //retry
							log.Printf("Reciever: Subscription failed: %s, will try again %v times", err.Error(), addr.Count)
							addr.Count--
							go func() {
								timer := time.NewTimer(addr.Interval)
								<-timer.C

								log.Printf("Reciever: Retrying subscription for %s", addr.Address)
								r.addressChan <- addr
							}()

						} else {
							log.Printf("Reciever: ERROR: %s", err.Error())
						}
						continue
					}

					//note that we already have a subscription so we won't run one again
					log.Printf("Receiver: subscription to %s added", addr.Address)
				} else {
					log.Printf("Reciever: Error - subscription channel closed")
					return
				}
			}
		}
	}()

	log.Printf("Reciever: reciever ready")
	//Start listening
	go func() {
		for {

			r.inChan <- sub.Read()

			//check for exit command
			select {
			case <-r.exitChan:
				wg.Done()
				return
			default:
				continue
			}
		}
	}()

	return nil
}

//Start starts the router on teh specified port, returns an address channel that may be used to add more subscriptions
func (r *Router) Start(routingGuide map[string][]string, wg sync.WaitGroup, channelSize int, addresses []string, retrySubscribeAttempts int, retrySubscribeInterval time.Duration, publisherPort string) error {
	//Build our channel
	r.inChan = make(chan common.Message, channelSize)
	r.outChan = make(chan common.Message, channelSize)
	r.exitChan = make(chan bool, 3)
	r.addressChan = make(chan subscriptionReq, 10)
	r.subscriptions = []string{}

	toListen := []string{}

	//start our publisher
	//func Publisher(messageChan <-chan common.Message, exit <-chan bool, port int, wg sync.WaitGroup) error {
	err := r.publisher(publisherPort, wg)
	if err != nil {
		return err
	}

	//start our router
	err = r.router(wg, routingGuide)
	if err != nil {
		return err
	}

	//build our list of things to listen to.
	for k, _ := range routingGuide {
		toListen = append(toListen, k)
	}

	err = r.reciever(toListen, wg)
	if err != nil {
		return err
	}

	//subscribe to all the addresses
	for _, addr := range addresses {
		r.Subscribe(addr, retrySubscribeAttempts, retrySubscribeInterval)
	}
	return nil
}

func (r *Router) Subscribe(address string, retryCount int, interval time.Duration) {
	r.addressChan <- subscriptionReq{Address: address, Count: retryCount, Interval: interval}
}

//The router takes messages in, relabels them according to the routing guide, and then outputs them.
//Note that the routing guide takes a first rule matched approach, so ensure that more specific rules are defined first
func (r *Router) router(wg sync.WaitGroup, routingGuide map[string][]string) error {
	log.Printf("Router: starting router")

	workingGuide := make(map[[24]byte][][24]byte)

	for k, v := range routingGuide {
		if len(k) > 24 {
			err := errors.New(fmt.Sprintf("Header %v is too long, max length is 24", k))
			wg.Done()
			return err
		}

		var from [24]byte
		var to [][24]byte

		copy(from[:], k)

		for _, val := range v {
			if len(val) > 24 {
				err := errors.New(fmt.Sprintf("Header %v is too long, max length is 24", val))
				wg.Done()
				return err
			}
			var cur [24]byte
			copy(cur[:], val)
			to = append(to, cur)
		}

		workingGuide[from] = to
	}

	go func() {
		for {
			select {
			case curEvent, ok := <-r.inChan:
				if ok {
					log.Printf("Event Recieved. Header: %s", curEvent.MessageHeader)

					if debug {
						log.Printf("Event: %s", curEvent.MessageBody)
					}

					for k, v := range workingGuide {
						if reflect.DeepEqual(k, curEvent.MessageHeader) {
							for i := range v {
								color.Set(color.FgBlue, color.Bold)
								log.Printf("Routing from %s to %s", curEvent.MessageHeader, v[i])
								color.Unset()
								r.outChan <- common.Message{MessageHeader: v[i], MessageBody: curEvent.MessageBody}
							}
							break
						}
					}

				} else {
					log.Printf("Router: In Channel closed, exiting")
					wg.Done()
					return
				}
			case <-r.exitChan:
				wg.Done()
				return
			}
		}
	}()
	return nil
}
