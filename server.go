package main

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/byuoitav/event-router-microservice/eventinfrastructure"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/xuther/go-message-router/router"
)

var dev bool

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
	RoutingTable[eventinfrastructure.Metrics] = []string{eventinfrastructure.Translator}
	RoutingTable[eventinfrastructure.UIFeature] = []string{eventinfrastructure.Room}

	router := router.Router{}

	err = router.Start(RoutingTable, wg, 1000, []string{}, 120, time.Second*3, port)
	if err != nil {
		log.Fatal(err)
	}

	server := echo.New()
	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.CORS())

	//	server.GET("/health", echo.WrapHandler(http.HandlerFunc(health.Check)))
	server.POST("/subscribe", eventinfrastructure.Subscribe)
	log.Printf("Waiting for new subscriptions")

	//	ip := eventinfrastructure.GetIP()
	pihn := os.Getenv("PI_HOSTNAME")
	if len(pihn) == 0 {
		log.Fatalf("PI_HOSTNAME is not set.")
	}
	//	values := strings.Split(strings.TrimSpace(pihn), "-")

	/*
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

					var s subscription.SubscribeRequest
					s.Address = ip + ".byu.edu:7000"
					s.PubAddress = ip + ".byu.edu:6999/subscribe"

					body, err := json.Marshal(s)
					if err != nil {
						log.Printf("[error] Failed to unmarshal subscription body: %s", err.Error())
					}

					for _, address := range addresses {
						log.Printf("Posting to %s", address)
						resp, err := http.Post("http://"+address, "application/json", bytes.NewBuffer(body))
						if err != nil {
							log.Printf("[error] Failed to post: %s", err.Error())
						}
						if resp != nil {
							defer resp.Body.Close()
							if resp.StatusCode == 200 {
								log.Printf("Post to %s successful", address)
							} else {
								log.Printf("[error] post to %s unsuccessful. Response:\n %v", address, resp)
							}
						}
					}
					return
				}
			}
		}()
	*/

	server.Start(":6999")
	wg.Wait()
}
