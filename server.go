package main

import (
	"sync"

	"github.com/byuoitav/zeromq-proxy-miroservice/helpers"
)

func main() {

	var wg sync.WaitGroup

	wg.Add(3)

	go helpers.PublishTest(wg)
	go helpers.RecieveTest(wg)
	go helpers.PublishTest1(wg)
	go helpers.PublishTest2(wg)

	wg.Wait()
}
