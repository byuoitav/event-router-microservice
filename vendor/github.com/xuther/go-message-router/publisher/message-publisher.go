package publisher

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xuther/go-message-router/common"
)

const debug = true
const pingPeriod = 30 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1048576, //one MB
	WriteBufferSize: 1048576, //one MB
}

type Publisher interface {
	Listen() error
	Write(common.Message) error
	Close()
}

type publisher struct {
	subscriptions    []*subscription
	port             string
	listener         *http.Server
	subscribeChan    chan *subscription
	UnsubscribeChan  chan *subscription
	writeQueueSize   int
	distributionChan chan common.Message
}

type subscription struct {
	pub        *publisher
	Connection *websocket.Conn
	WriteQueue chan common.Message
}

func NewPublisher(port string, writeQueueSize int, subscribeChanSize int) (Publisher, error) {
	return &publisher{
		port:             port,
		writeQueueSize:   writeQueueSize,
		subscribeChan:    make(chan *subscription, subscribeChanSize),
		UnsubscribeChan:  make(chan *subscription, subscribeChanSize),
		distributionChan: make(chan common.Message, writeQueueSize),
	}, nil
}

func (p *publisher) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("There was a problem accepting a connection from %v Error: %v", r.RemoteAddr, err.Error())
		return
	}

	p.subscribeChan <- &subscription{Connection: conn, WriteQueue: make(chan common.Message, p.writeQueueSize), pub: p}
}

//Listen will start a TCP listener bound to the port in the publisher in a separate go routine. Connections are added to the subscriptions slice
//Use the Write function to send a message to all subscribers
func (p *publisher) Listen() error {

	if len(p.port) == 0 {
		return errors.New("The publisher must be initialized with a port")
	}

	//Start the membership routine
	p.runMembership()

	//Start the broadcast routine
	p.runBroadcaster()

	//we can't use the default server mux
	serverMux := http.NewServeMux()
	serverMux.Handle("/subscribe", p)

	srv := &http.Server{
		Addr:    ":" + p.port,
		Handler: serverMux,
	}

	p.listener = srv
	err := srv.ListenAndServe()

	if err != nil {
		log.Printf("ERROR starting publisher listener: %v", err.Error())
		return err
	}

	return nil
}

func (p *publisher) Close() {

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if p.listener != nil {
		p.listener.Shutdown(ctx)
	}
}

func (p *publisher) Write(event common.Message) error {
	if debug {
		log.Printf("sending a message to the distribution channel")
	}

	p.distributionChan <- event
	return nil
}

func (p *publisher) runBroadcaster() error {
	go func() {
		d := 0
		log.Printf("Starting distribution Channel")
		for {
			curMessage := <-p.distributionChan
			if debug {
				log.Printf("Received a message in distribution channel. Distributing...")
				log.Printf("There are %v subscriptions to send to", len(p.subscriptions))
			}
			for i := range p.subscriptions {
				select {
				case p.subscriptions[i].WriteQueue <- curMessage:

					if debug {
						log.Printf("Sending a message to: %v", p.subscriptions[i].Connection.RemoteAddr().String())
					}
				default:
					d++
					if d%100 == 0 {
						log.Printf("%v discarded.", d)
					}
				}
			}
		}
	}()
	return nil
}

//runMembership handles adding and removing from the membership array
func (p *publisher) runMembership() {
	go func() {
		for {
			select {

			//Add a subscription
			case subscription := <-p.subscribeChan:
				log.Printf("Subscription receieved for %s", subscription.Connection.RemoteAddr().String())
				p.subscriptions = append(p.subscriptions, subscription)
				subscription.StartWriter()
				break

			//Remove a subscription
			case subscription := <-p.UnsubscribeChan:
				for i := range p.subscriptions {
					if p.subscriptions[i] == subscription {
						//This is the go sanctioned way of deleting an item from an array that contains pointers for garbage collection.
						//Note that due to the format of slice headers the calculation fo len(slice) is constant time.
						p.subscriptions[i].Connection.Close() //make sure the connection is closed, if it's already closed, don't worry about it
						p.subscriptions[i] = p.subscriptions[len(p.subscriptions)-1]
						p.subscriptions[len(p.subscriptions)-1] = nil
						p.subscriptions = p.subscriptions[:len(p.subscriptions)-1]
						break
					}
				}
				break

			}

		}

	}()
}

//StartWriter runs a writer routine for the subscription, listening for messages to send
func (s *subscription) StartWriter() {
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			s.Connection.Close()
		}()
		for {
			select {
			case toWrite := <-s.WriteQueue:
				if debug {
					log.Printf("Sending Message to %s", s.Connection.RemoteAddr().String())
				}

				err := s.Connection.WriteJSON(toWrite)
				if err != nil {
					log.Printf("ERROR: there was a problem with the connection to client: %s. Message: %s", s.Connection.RemoteAddr().String(), err.Error())

					log.Printf("Connection closed : %s", s.Connection.RemoteAddr().String())
					s.pub.UnsubscribeChan <- s //end the connection to be removed and closed
					return                     //End
				}
			case <-ticker.C:
				log.Printf("Sending Ping message")
				if err := s.Connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					s.pub.UnsubscribeChan <- s //end the connection to be removed and closed
					return

				}
			}
		}
	}()
}
