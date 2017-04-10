package helpers

import (
	"log"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
)

func PublishTest2(wg sync.WaitGroup) error {

	publisher, err := zmq.NewSocket(zmq.PUB)

	if err != nil {
		return err
	}

	defer publisher.Close()

	publisher.Bind("tcp://*:6967")

	for i := 0; i < 1000; i++ {
		publisher.Send("B", zmq.SNDMORE)
		publisher.Send("a", 0)

		log.Printf("Sending...")
		time.Sleep(time.Second / 10)
	}

	publisher.Send("B", zmq.SNDMORE)
	publisher.Send("We would like to see this", 0)

	wg.Done()
	return nil
}

func PublishTest(wg sync.WaitGroup) error {

	publisher, err := zmq.NewSocket(zmq.PUB)

	if err != nil {
		return err
	}

	defer publisher.Close()

	publisher.Bind("tcp://*:6969")

	for i := 0; i < 1000; i++ {
		publisher.Send("B", zmq.SNDMORE)
		publisher.Send("We would like to see this", 0)

		log.Printf("Sending...")
		time.Sleep(time.Second / 10)
	}

	publisher.Send("B", zmq.SNDMORE)
	publisher.Send("We would like to see this", 0)

	wg.Done()
	return nil
}

func PublishTest1(wg sync.WaitGroup) error {

	publisher1, err := zmq.NewSocket(zmq.PUB)

	if err != nil {
		return err
	}

	defer publisher1.Close()

	publisher1.Bind("tcp://*:6968")

	for i := 0; i < 10; i++ {

		log.Printf("Sending...")
		publisher1.Send("B", zmq.SNDMORE)
		publisher1.Send("Yo Home Dog", 0)
		time.Sleep(time.Second / 10)
	}

	wg.Done()
	return nil
}

func RecieveTest(wg sync.WaitGroup) error {

	subscriber, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		return err
	}

	defer subscriber.Close()

	err = subscriber.Connect("tcp://localhost:6968")
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
		return err
	}
	err = subscriber.Connect("tcp://localhost:6969")
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
		return err
	}

	err = subscriber.SetSubscribe("B")
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
		return err
	}

	i := 0

	for {

		if i == 100 {
			err = subscriber.Connect("tcp://localhost:6967")
			if err != nil {
				log.Printf("ERROR: %s", err.Error())
				return err
			}
		}
		i++

		address, _ := subscriber.Recv(0)

		contents, _ := subscriber.Recv(0)
		log.Printf("%s: %s", address, contents)

	}

	wg.Done()

	return nil

}
