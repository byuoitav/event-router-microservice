package eventinfrastructure

import (
	"encoding/json"
	"log"

	"github.com/xuther/go-message-router/common"
	"github.com/xuther/go-message-router/publisher"
)

type Publisher struct {
	publisher publisher.Publisher
	writeChan chan common.Message
	Port      string
}

// users of this publisher struct can write messages into the write channel, and they will be published
func NewPublisher(port string) *Publisher {
	var p Publisher
	var err error

	p.Port = port
	p.publisher, err = publisher.NewPublisher(p.Port, 1001, 10)
	if err != nil {
		log.Fatalf("[error] Failed to create publisher. error: %s", err.Error())
	}

	// start listening on the channel to publish
	p.writeChan = make(chan common.Message)

	go p.publisher.Listen()
	log.Printf("[publisher] Publisher successfully started on port %s. Publish away!", port)

	// write things that come on the channel
	go func() {
		for {
			select {
			case message, ok := <-p.writeChan:
				if !ok {
					log.Fatalf("[error] publisher write channel closed")
				}

				log.Printf("[publisher] Publishing message: %s", message)
				err = p.publisher.Write(message)
				if err != nil {
					log.Printf("[error] error publishing message: %s", err.Error())
				}
			}
		}
	}()

	return &p
}

func (p *Publisher) PublishEvent(e Event, eventType string) error {
	toSend, err := json.Marshal(&e)
	if err != nil {
		return err
	}

	header := [24]byte{}
	copy(header[:], []byte(eventType))

	p.writeChan <- common.Message{MessageHeader: header, MessageBody: toSend}
	return nil
}

func (p *Publisher) PublishCommonMessage(m common.Message) {
	p.writeChan <- m
}
