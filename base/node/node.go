package node

import (
	"fmt"
	"log"
	"time"

	"github.com/byuoitav/event-router-microservice/base"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the router.
	pingWait = 90 * time.Second

	// Interval to wait between retry attempts
	retryInterval = 3 * time.Second
)

type Node struct {
	Name          string
	Conn          *websocket.Conn
	WriteQueue    chan base.Message
	ReadQueue     chan base.Message
	RouterAddress string
	filters       map[string]bool
	readDone      chan bool
	writeDone     chan bool
	lastPingTime  time.Time
}

func (n *Node) GetState() string {
	toReturn := ""
	toReturn += "Name: \t\t\t " + n.Name + "\n"
	toReturn += fmt.Sprintf("Router: \t\t\t %v \n", n.RouterAddress)
	toReturn += fmt.Sprintf("Connection: \t\t\t %v -> %v \n", n.Conn.LocalAddr().String, n.Conn.RemoteAddr().String())
	toReturn += fmt.Sprintf("Filters:\n")
	for filter := range n.filters {
		toReturn += fmt.Sprintf("\t\t\t\t\t %v\n", filter)
	}
	toReturn += fmt.Sprintf("Last Ping Time: \t\t\t %v\n", n.lastPingTime.Format(time.RFC3339))

	return toReturn
}

func (n *Node) Start(RouterAddress string, filters []string, name string) error {

	log.Printf(color.HiGreenString("Starting EventNode. Connecting to router: %v", RouterAddress))

	n.RouterAddress = RouterAddress
	n.ReadQueue = make(chan base.Message, 4096)
	n.WriteQueue = make(chan base.Message, 4096)
	n.readDone = make(chan bool)
	n.writeDone = make(chan bool)
	n.filters = make(map[string]bool)
	n.Name = name

	for _, f := range filters {
		n.filters[f] = true
	}

	err := n.openConnection()
	if err != nil {

		n.readDone <- true
		n.writeDone <- true

		go n.retryConnection()
		return nil
	}

	log.Printf(color.HiGreenString("Starting pumps..."))
	go n.readPump()
	go n.writePump()

	return nil

}

func (n *Node) openConnection() error {
	//open connection to the router
	var dialer *websocket.Dialer

	conn, _, err := dialer.Dial(fmt.Sprintf("ws://%s/subscribe", n.RouterAddress), nil)
	if err != nil {
		log.Printf(color.HiRedString("There was a problem establishing the websocket with %v : %v", n.RouterAddress, err.Error()))
		return err
	}
	n.Conn = conn

	return nil
}

func (n *Node) retryConnection() {
	log.Printf(color.HiMagentaString("[retry] Retrying connection, waiting for read and write pump to close before starting."))
	//wait for read to say i'm done.
	<-n.readDone
	log.Printf(color.HiMagentaString("[retry] Read pump closed"))

	//wait for write to be done.
	<-n.writeDone
	log.Printf(color.HiMagentaString("[retry] Write pump closed"))
	log.Printf(color.HiMagentaString("[retry] Retrying connection"))

	//we retry
	err := n.openConnection()

	for err != nil {
		log.Printf(color.HiMagentaString("[retry] Retry failed, trying again in 3 seconds."))
		time.Sleep(retryInterval)
		err = n.openConnection()
	}
	//start the pumps again
	log.Printf(color.HiGreenString("[Retry] Retry success. Starting pumps"))
	go n.readPump()
	go n.writePump()

}

func (n *Node) readPump() {

	defer func() {
		n.Conn.Close()
		log.Printf(color.HiRedString("Connection to router %v is dying.", n.RouterAddress))

		n.readDone <- true
	}()

	n.Conn.SetPingHandler(
		func(string) error {
			log.Printf("Ping...is a name\n")
			n.Conn.SetReadDeadline(time.Now().Add(pingWait))
			n.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))

			//debugging purposes
			n.lastPingTime = time.Now()

			return nil
		})

	n.Conn.SetReadDeadline(time.Now().Add(pingWait))

	for {
		var message base.Message
		err := n.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}

		_, ok := n.filters[message.MessageHeader]
		if !ok {
			continue
		}

		//we need to check against the list of accepted values
		n.ReadQueue <- message
	}
}

func (n *Node) writePump() {

	defer func() {
		n.Conn.Close()
		log.Printf(color.HiRedString("Connection to router %v is dying. Trying to resurrect.", n.RouterAddress))
		n.writeDone <- true

		//try to reconnect
		n.retryConnection()
	}()

	for {
		select {
		case message, ok := <-n.WriteQueue:
			if !ok {
				n.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(writeWait))
				return
			}

			err := n.Conn.WriteJSON(message)
			if err != nil {
				return
			}
		case <-n.readDone:
			//put it back in
			n.readDone <- true
			return
		}
	}
}

func (n *Node) Write(message base.Message) error {
	n.WriteQueue <- message
	return nil

}

func (n *Node) Read() base.Message {
	msg := <-n.ReadQueue
	return msg
}

func (n *Node) Close() {

}
