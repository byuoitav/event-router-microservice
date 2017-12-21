package router

import (
	"log"

	"github.com/byuoitav/event-router-microservice/base"
	"github.com/fatih/color"
)

type Router struct {
	unregister        chan *Subscription
	register          chan *Subscription
	inChan            chan base.Message
	outChan           chan base.Message
	subscriptions     map[*Subscription]bool
	routingTable      map[string][]string
	routerConnections map[*RouterBridge]bool
}

func (r *Router) StartRouter(RoutingTable map[string][]string) error {

	for {
		select {
		case sub := <-r.register:
			r.subscriptions[sub] = true

		case sub := <-r.unregister:
			if _, ok := r.subscriptions[sub]; ok {
				delete(r.subscriptions, sub)
				close(sub.send)
			}
		case message := <-r.inChan:
			//we need to run it through the routing table
			r.route(message)
		}
	}
}

//ConnectToRouters takes a list of peer routers to connect to.
func (r *Router) ConnectToRouters(peerAddresses []string) error {

	//build our filters
	filters := []string{}

	for k := range r.routingTable {
		filters = append(filters, k)
	}

	for _, addr := range peerAddresses {

		bridge, err := StartBridge(addr, filters, r)
		if err != nil {
			log.Printf(color.HiRedString("Could not establish connection to the peer %v", addr))
			continue
		}

		go bridge.ReadPassthrough()
		r.routerConnections[bridge] = true
	}

	return nil
}

func (r *Router) Stop() error {
	return nil
}

func NewRouter() *Router {
	return &Router{
		inChan:        make(chan base.Message, 1024),
		outChan:       make(chan base.Message, 1024),
		register:      make(chan *Subscription),
		unregister:    make(chan *Subscription),
		subscriptions: make(map[*Subscription]bool),
	}
}

func (r *Router) route(message base.Message) {

	headers, ok := r.routingTable[message.MessageHeader]
	if !ok {
		//it's not a message we care about
		return
	}

	for _, newHeader := range headers {
		for sub := range r.subscriptions {
			select {
			case sub.send <- base.Message{MessageBody: message.MessageBody, MessageHeader: newHeader}:
			default:
				close(sub.send)
				delete(r.subscriptions, sub)
			}
		}

		for routerConn := range r.routerConnections {
			routerConn.WritePassthrough(base.Message{MessageBody: message.MessageBody, MessageHeader: newHeader})
		}

	}
}

func (r *Router) GetInfo() string {

	toReturn := ""

	for v := range r.routerConnections {
		toReturn += v.Node.GetState() + "\n\n"
	}

	return toReturn
}
