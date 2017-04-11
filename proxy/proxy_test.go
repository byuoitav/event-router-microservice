package proxy

import (
	"sync"
	"testing"

	"github.com/byuoitav/zeromq-proxy-miroservice/zeromq"
)

func TestRouter(t *testing.T) {

	inChan := make(chan zeromq.Event, 5)
	outChan := make(chan zeromq.Event, 5)
	stopChan := make(chan bool, 2)

	var wg sync.WaitGroup
	wg.Add(1)

	go Router(inChan, outChan, stopChan, wg)

	//Test every entry in the table
	LocalAPI := zeromq.Event{zeromq.LocalAPI, "b"}
	TransmitAPI := zeromq.Event{zeromq.TransmitAPI, "b"}
	External := zeromq.Event{zeromq.External, "b"}
	LocalTransmit := zeromq.Event{zeromq.LocalTransmit, "b"}

	inChan <- LocalAPI
	test := <-outChan

	if test.Envelope != zeromq.TransmitAPI {
		t.Fail()
	}

	inChan <- TransmitAPI
	test = <-outChan

	if test.Envelope != zeromq.LocalTransmit {
		t.Fail()
	}

	inChan <- External
	test = <-outChan

	if test.Envelope != zeromq.LocalTransmit {
		t.Fail()
	}

	inChan <- LocalTransmit
	inChan <- LocalAPI
	test = <-outChan

	if test.Envelope != zeromq.TransmitAPI {
		t.Fail()
	}

}
