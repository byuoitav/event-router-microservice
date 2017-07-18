package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
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

var dev bool

func main() {
	var wg sync.WaitGroup
	var err error

	GetIP()

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

	// get all the devices with the eventrouter role
	subscription.Hostname = os.Getenv("PI_HOSTNAME")
	if len(subscription.Hostname) == 0 {
		log.Fatalf("[error] PI_HOSTNAME is not set.")
	}
	log.Printf("PI_HOSTNAME = %s", subscription.Hostname)
	values := strings.Split(strings.TrimSpace(subscription.Hostname), "-")

	devhn := os.Getenv("DEVELOPMENT_HOSTNAME")
	dev = false
	if len(devhn) != 0 {
		dev = true
		log.Printf("Development machine. Using hostname %s", devhn)
		subscription.Hostname = devhn
	}

	// overwrite hostname with ip address
	subscription.Hostname = ip
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
						if strings.EqualFold(device.GetFullName(), subscription.Hostname) {
							continue
						}
					}
					addresses = append(addresses, device.Address+":6999/subscribe")
				}

				var s subscription.SubscribeRequest
				s.Address = subscription.Hostname + ".byu.edu:7000"
				s.PubAddress = subscription.Hostname + ".byu.edu:6999/subscribe"
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
							log.Printf("[error] post to %s unsuccessful. Response:\n %s", address, resp)
						}
					}
				}
				return
			}
		}
	}()

	server.Start(":6999")
	wg.Wait()
}

func GetIP() string {
	var ip net.IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return err.Error()
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && strings.Contains(address.String(), "/24") {
			ip, _, err = net.ParseCIDR(address.String())
			if err != nil {
				log.Fatalf("[error] %s", err.Error())
			}
		}
	}

	if ip == nil {
		log.Fatalf("Failed to find an non-loopback IP Address.")
	}

	log.Printf("My IP address is %s", ip)

	return string(ip)
}

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")

	return localAddr[0:idx]
}
