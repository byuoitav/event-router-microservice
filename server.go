package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
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
	go server.Start(":6999")

	// get all the devices with the eventrouter role
	hostname := os.Getenv("PI_HOSTNAME")
	ip := os.Getenv("IP_ADDR")
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

				addresses := []string{}
				for _, device := range devices {
					if strings.EqualFold(device.GetFullName(), hostname) {
						continue
					}
					// hit each of these addresses subscription endpoint once
					// to try and create a two-way subscription between the event routers
					addresses = append(addresses, "http://"+device.Address+":6999/subscribe")
				}

				var s subscription.SubscribeRequest
				s.Address = ip + ":7000"
				s.PubAddress = ip + ":6999/subscribe"
				body, err := json.Marshal(s)
				if err != nil {
					log.Printf("[error] %s", err.Error())
				}
				bodyInBytes := bytes.NewBuffer(body)

				for _, address := range addresses {
					log.Printf("Posting to %s", address)
					resp, err := http.Post(address, "application/json", bodyInBytes)
					if err != nil {
						log.Printf("[error] %s", err.Error())
					}
					if resp != nil {
						log.Printf("response: %s", resp)
					}
				}

				return
			}
		}
	}()

	wg.Wait()
}
