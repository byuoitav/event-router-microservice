package main

import (
	"log"
	"os"
	"strings"
	"sync"

	"github.com/byuoitav/av-api/dbo"
	"github.com/byuoitav/event-router-microservice/tags"
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
		addresses = append(addresses, device.Address+":7000")
	}

	//subscribe to the av-api and the event translator on the local pi
	addresses = append(addresses, "localhost:7001", "localhost:7002")

	RoutingTable := make(map[string][]string)
	RoutingTable[tags.LocalAPI] = []string{tags.TransmitAPI}
	RoutingTable[tags.TransmitAPI] = []string{tags.LocalTransmit}
	RoutingTable[tags.External] = []string{tags.LocalTransmit}

	err := router.Start(RoutingTable, wg, 1000, addresses, 120, "7000")
	if err != nil {
		fmt.Error(err)
	}

	wg.Wait()
}
