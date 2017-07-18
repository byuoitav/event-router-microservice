package eventinfrastructure

import (
	"log"

	"github.com/xuther/go-message-router/common"
	"github.com/xuther/go-message-router/publisher"
)

type Publisher struct {
	publisher publisher.Publisher
	WriteChan chan common.Message
}

// users of this publisher struct can write messages into the write channel, and they will be published
func (p Publisher) Start(port string) {
	pub, err := publisher.NewPublisher(port, 1001, 10)
	if err != nil {
		log.Fatalf("[error] Failed to create publisher. error: %s", err.Error())
	}

	p.publisher = pub

	// start listening on the channel to publish
	p.WriteChan = make(chan common.Message)

	go p.publisher.Listen()

	// write things that come on the channel
	for {
		select {
		case message, ok := <-p.WriteChan:
			if !ok {
				log.Printf("[error] publisher write channel closed")
				return
			}
			log.Printf("[publisher] Publishing message: %s", message)
			err = p.publisher.Write(message)
			if err != nil {
				log.Printf("[error] error publishing message: %s", err.Error())
			}
		}
	}
}
