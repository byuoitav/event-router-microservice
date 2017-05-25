package main

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/byuoitav/av-api/dbo"
	"github.com/byuoitav/event-router-microservice/eventinfrastructure"
	"github.com/byuoitav/event-router-microservice/handlers"
	"github.com/byuoitav/go-message-router/router"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var retryCount = 60

func main() {

	var wg sync.WaitGroup

	wg.Add(3)
	port := "7000"

	//Get all the devices with role "Event Router"
	hostname := os.Getenv("PI_HOSTNAME")
	values := strings.Split(strings.TrimSpace(hostname), "-")
	devices, err := dbo.GetDevicesByBuildingAndRoomAndRole(values[0], values[1], "EventRouter")

	if err != nil {
		for retryCount > 0 {
			retryCount--
			devices, err = dbo.GetDevicesByBuildingAndRoomAndRole(values[0], values[1], "EventRouter")
			if err != nil && retryCount > 0 {

				log.Printf("Connection to the Configuration DB failed, retrying in 2 seconds, will retry %v more times", retryCount)
				timer := time.NewTimer(2 * time.Second)
				<-timer.C
				continue
			} else if err != nil {
				log.Fatal(err.Error())
			} else if err == nil {
				break
			}
		}
	}
	log.Printf("Connection to the DB established")

	addresses := []string{}
	for _, device := range devices {
		if strings.EqualFold(device.GetFullName(), hostname) {
			continue
		}

		addresses = append(addresses, device.Address+":7000")
	}

	//subscribe to the av-api and the event translator on the local pi
	addresses = append(addresses, "localhost:7001", "localhost:7002")

	RoutingTable := make(map[string][]string)
	RoutingTable[eventinfrastructure.Room] = []string{eventinfrastructure.UI}
	RoutingTable[eventinfrastructure.APISuccess] = []string{
		eventinfrastructure.Translator,
		eventinfrastructure.UI,
		eventinfrastructure.Room,
	}
	RoutingTable[eventinfrastructure.External] = []string{eventinfrastructure.UI}
	RoutingTable[eventinfrastructure.APIError] = []string{eventinfrastructure.UI, eventinfrastructure.Translator}

	r := router.Router{}

	err = r.Start(RoutingTable, wg, 1000, addresses, 120, time.Second*3, port)
	if err != nil {
		log.Fatal(err)
	}

	server := echo.New()
	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.CORS())

	//	server.GET("/health", echo.WrapHandler(http.HandlerFunc(health.Check)))
	server.GET("/subscribe", handlers.Subscribe)

	server.Start("70000")

	wg.Wait()
}
