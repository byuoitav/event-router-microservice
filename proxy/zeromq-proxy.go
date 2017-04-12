package proxy

import (
	"fmt"
	"log"
	"sync"

	"github.com/byuoitav/zeromq-proxy-miroservice/zeromq"
	zmq "github.com/pebbe/zmq4"
)

//Publish starts a ZMQ publisher on the port provided, and listens for events to send on the eventChan.
//Will run until an exit signal is sent down the exit channel
func Publish(eventChan <-chan zeromq.Event, exit <-chan bool, port int, wg sync.WaitGroup) error {
	publisher, err := zmq.NewSocket(zmq.PUB)

	if err != nil {
		return err
	}

	defer publisher.Close()

	addr := fmt.Sprintf("tcp://*:%v", port)
	err = publisher.Bind(addr)
	if err != nil {
		log.Printf("Error binding to publisher port: %s", err.Error())
		return err
	}

	for {
		select {
		case event := <-eventChan:
			sendEvent(event, publisher)
		case <-exit:
			wg.Done()
			return nil
		}
	}

}

//sendEvent sends the event down the provided socket
//NOTE: zmq sockets are not threadsafe, this should never be invoked via go sendzeromq.Event, only from a routine with a dedicated
//zmq socket.
func sendEvent(event zeromq.Event, pub *zmq.Socket) error {

	pub.Send(event.Envelope, zmq.SNDMORE)
	pub.Send(event.Data, 0)

	return nil
}

func setSubscribe(sub *zmq.Socket) error {
	toSubscribe := [...]string{
		zeromq.LocalAPI,
		zeromq.TransmitAPI,
		zeromq.LocalTransmit,
		zeromq.External,
	}

	for _, str := range toSubscribe {
		err := sub.SetSubscribe(str)
		if err != nil {
			log.Printf("Error settings subscription filters: %s", err.Error())
			return err
		}
	}

	return nil

}

//Receive starts a listener on all ports passed in, On event receipt sends the event to the router.
func Recieve(eventChan chan<- zeromq.Event, exit <-chan bool, addresses []string, wg sync.WaitGroup) error {

	subscriber, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		wg.Done()
		return err
	}

	defer subscriber.Close()

	for _, addr := range addresses {
		log.Printf("Starting connection with %s", addr)
		err = subscriber.Connect("tcp://" + addr)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			wg.Done()
			return err
		}

	}

	//Subscribe to all of the event types
	err = setSubscribe(subscriber)
	if err != nil {
		wg.Done()
		return err
	}

	//Start listening
	for {
		envelope, err := subscriber.Recv(0)
		if err != nil {
			log.Printf("Error receiving envelope: %s", err.Error())
		}
		data, err := subscriber.Recv(0)
		if err != nil {
			log.Printf("Error receiving envelope: %s", err.Error())
		}
		eventChan <- zeromq.Event{Data: data, Envelope: envelope}

		//check for exit command
		select {
		case <-exit:
			wg.Done()
			return nil
		default:
			continue
		}
	}
}

/*
Router takes in the events, and relabels them according to the routing table
          In      |       Out
-------------------------------------
      LocalAPI    |    TransmitAPI
    Transmit API  |    Local Transmit
      External	  |    Local Transmit


For a better description of the event types see the wiki and the definition of the constants.
*/
func Router(inChan <-chan zeromq.Event, outChan chan<- zeromq.Event, exit <-chan bool, wg sync.WaitGroup) error {
	log.Printf("Starting router")

	for {
		select {
		case curEvent, ok := <-inChan:
			if ok {

				//Run through our routing rules, translate the envelope, and send it down the out channel
				switch curEvent.Envelope {
				case zeromq.LocalAPI:
					curEvent.Envelope = zeromq.TransmitAPI
					outChan <- curEvent
				case zeromq.TransmitAPI:
					curEvent.Envelope = zeromq.LocalTransmit
					outChan <- curEvent
				case zeromq.External:
					curEvent.Envelope = zeromq.LocalTransmit
					outChan <- curEvent
				default:
					break
				}

			} else { // the channel was closed
				return nil
			}
		case <-exit:
			log.Printf("exit command received, exiting router.")
			return nil
		}
	}
}
