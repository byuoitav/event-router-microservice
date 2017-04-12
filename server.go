package main

import (
	"log"
	"os"
	"strings"
	"sync"

	"github.com/byuoitav/av-api/dbo"
	"github.com/byuoitav/zeromq-proxy-miroservice/proxy"
	"github.com/byuoitav/zeromq-proxy-miroservice/zeromq"
)

func main() {

	var wg sync.WaitGroup

	wg.Add(3)
	port := 7000

	//Get all the devices with role "Event Router"
	hostname := os.Getenv("PI_HOSTNAME")
	values := strings.Split(strings.TrimSpace(hostname), "-")
	devices, err := dbo.GetDevicesByBuildingAndRoomAndRole(values[0], values[1], "EventRouter")
	if err != nil {
		log.Fatal(err.Error())
	}

	addresses := []string{}
	for _, device := range devices {
		addresses = append(addresses, device.Address)
	}

	inChan := make(chan zeromq.Event, 100)
	outChan := make(chan zeromq.Event, 100)
	exitChan := make(chan bool, 3)

	log.Printf("Starting publisher")
	go proxy.Publish(outChan, exitChan, port, wg)
	log.Printf("Starting router")
	go proxy.Router(inChan, outChan, exitChan, wg)
	log.Printf("Starting Receiver")
	go proxy.Recieve(inChan, exitChan, addresses, wg)

	wg.Wait()
}
