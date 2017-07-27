package main

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/byuoitav/av-api/dbo"
	"github.com/byuoitav/event-router-microservice/eventinfrastructure"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var dev bool

func main() {
	var wg sync.WaitGroup

	wg.Add(3)
	port := "7000"

	RoutingTable := make(map[string][]string)
	RoutingTable[eventinfrastructure.Room] = []string{eventinfrastructure.UI}
	RoutingTable[eventinfrastructure.APISuccess] = []string{
		eventinfrastructure.Translator,
		eventinfrastructure.UI,
		eventinfrastructure.Room,
	}
	RoutingTable[eventinfrastructure.External] = []string{eventinfrastructure.UI}
	RoutingTable[eventinfrastructure.APIError] = []string{eventinfrastructure.UI, eventinfrastructure.Translator}
	RoutingTable[eventinfrastructure.Metrics] = []string{eventinfrastructure.Translator}
	RoutingTable[eventinfrastructure.UIFeature] = []string{eventinfrastructure.Room}

	router := eventinfrastructure.NewRouter(RoutingTable, wg, port)

	server := echo.New()
	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.CORS())

	//	server.GET("/health", echo.WrapHandler(http.HandlerFunc(health.Check)))
	server.POST("/subscribe", router.HandleRequest)

	ip := eventinfrastructure.GetIP()
	pihn := os.Getenv("PI_HOSTNAME")
	if len(pihn) == 0 {
		log.Fatalf("PI_HOSTNAME is not set.")
	}
	values := strings.Split(strings.TrimSpace(pihn), "-")

	go func() {
		for {
			devices, err := dbo.GetDevicesByBuildingAndRoomAndRole(values[0], values[1], "EventRouter")
			if err != nil {
				log.Printf("[error] Connecting to the Configuration DB failed, retrying in 5 seconds.")
				time.Sleep(5 * time.Second)
			} else {
				log.Printf("Connection to the Configuration DB established.")

				addresses := []string{}
				for _, device := range devices {
					if !dev {
						if strings.EqualFold(device.GetFullName(), pihn) {
							continue
						}
					}
					addresses = append(addresses, device.Address+":6999/subscribe")
				}

				var cr eventinfrastructure.ConnectionRequest
				cr.PublisherAddr = ip + ".byu.edu:7000"
				cr.SubscriberEndpoint = ip + ".byu.edu:6999/subscribe"

				for _, address := range addresses {
					eventinfrastructure.SendConnectionRequest("http://"+address, cr, false)
				}
				return
			}
		}
	}()

	server.Start(":6999")
	wg.Wait()
}
