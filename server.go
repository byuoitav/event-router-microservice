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
	"github.com/byuoitav/event-router-microservice/subscription"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/xuther/go-message-router/router"
)

var retryCount = 60

func main() {
	var wg sync.WaitGroup
	var err error

	wg.Add(3)
	port := "7000"

	// get all the devices with the eventrouter role
	hostname := os.Getenv("PI_HOSTNAME")
	if len(hostname) == 0 {
		log.Fatalf("[error] PI_HOSTNAME is not set.")
	}
	values := strings.Split(strings.TrimSpace(hostname), "-")
	go func() {
		for {
			devices, err := dbo.GetDevicesByBuildingAndRoomAndRole(values[0], values[1], "EventRouter")
			if err != nil {
				log.Printf("[error] Connecting to the Configuration DB failed, retrying in 5 seconds.")
				time.Sleep(5 * time.Second)
			} else {
				log.Printf("Connection to the Configuration DB established.")

				for _, device := range devices {
					if strings.EqualFold(device.GetFullName(), hostname) {
						continue
					}
					// hit each of these addresses subscription endpoint once
					// to try and create a two-way subscription between the event routers
				}
				return
			}
		}
	}()

	RoutingTable := make(map[string][]string)
	RoutingTable[eventinfrastructure.Room] = []string{eventinfrastructure.UI}
	RoutingTable[eventinfrastructure.APISuccess] = []string{
		eventinfrastructure.Translator,
		eventinfrastructure.UI,
		eventinfrastructure.Room,
	}
	RoutingTable[eventinfrastructure.External] = []string{eventinfrastructure.UI}
	RoutingTable[eventinfrastructure.APIError] = []string{eventinfrastructure.UI, eventinfrastructure.Translator}

	subscription.R = router.Router{}

	err = subscription.R.Start(RoutingTable, wg, 1000, []string{}, 120, time.Second*3, port)
	if err != nil {
		log.Fatal(err)
	}

	server := echo.New()
	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.CORS())

	//	server.GET("/health", echo.WrapHandler(http.HandlerFunc(health.Check)))
	server.POST("/subscribe", handlers.Subscribe)

	log.Printf("Waiting for new subscriptions")
	server.Start(":6999")

	wg.Wait()
}
