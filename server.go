package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/byuoitav/av-api/dbo"
	"github.com/byuoitav/device-monitoring-microservice/statusinfrastructure"
	"github.com/byuoitav/event-router-microservice/eventinfrastructure"
	"github.com/fatih/color"
	"github.com/jessemillar/health"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	defer color.Unset()
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

	RoutingTable[eventinfrastructure.TestStart] = []string{eventinfrastructure.TestPleaseReply}
	RoutingTable[eventinfrastructure.TestPleaseReply] = []string{eventinfrastructure.TestExternal}
	RoutingTable[eventinfrastructure.TestExternalReply] = []string{eventinfrastructure.TestReply}
	RoutingTable[eventinfrastructure.TestReply] = []string{eventinfrastructure.TestEnd}

	var nodes []string
	addrs := strings.Split(os.Getenv("EVENT_NODE_ADDRESSES"), ",")
	for _, addr := range addrs {
		nodes = append(nodes, addr)
	}

	// create the router
	router := eventinfrastructure.NewRouter(RoutingTable, wg, port, nodes)

	server := echo.New()
	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.CORS())

	server.GET("/health", echo.WrapHandler(http.HandlerFunc(health.Check)))
	server.GET("/mstatus", GetStatus)
	server.POST("/subscribe", router.HandleRequest)

	go SubscribeOutsidePi(router)

	server.Start(":6999")
	wg.Wait()
}

func SubscribeOutsidePi(router *eventinfrastructure.Router) {
	devhn := os.Getenv("DEVELOPMENT_HOSTNAME")
	if len(devhn) > 0 {
		color.Set(color.FgYellow)
		log.Printf("Development machine. Using hostname %s", devhn)
		color.Unset()
	}

	ip := eventinfrastructure.GetIP()

	pihn := os.Getenv("PI_HOSTNAME")
	if len(pihn) == 0 {
		log.Fatalf("PI_HOSTNAME is not set.")
	}
	values := strings.Split(strings.TrimSpace(pihn), "-")

	for {
		devices, err := dbo.GetDevicesByBuildingAndRoomAndRole(values[0], values[1], "EventRouter")
		if err != nil {
			color.Set(color.FgRed)
			log.Printf("Connecting to the Configuration DB failed, retrying in 5 seconds.")
			color.Unset()
			time.Sleep(5 * time.Second)
		} else {
			color.Set(color.FgYellow, color.Bold)
			log.Printf("Connection to the Configuration DB established.")
			color.Unset()

			addresses := []string{}
			for _, device := range devices {
				if len(devhn) == 0 {
					if strings.EqualFold(device.GetFullName(), pihn) {
						continue
					}
				}
				addresses = append(addresses, device.Address+":6999/subscribe")
			}

			var cr eventinfrastructure.ConnectionRequest
			cr.PublisherAddr = ip + ":7000"
			cr.SubscriberEndpoint = fmt.Sprintf("http://%s:6999/subscribe", ip)

			for _, address := range addresses {
				split := strings.Split(address, ":")
				host, err := net.LookupHost(split[0])
				if err != nil {
					log.Printf("error %s", err.Error())
				}
				color.Set(color.FgYellow, color.Bold)
				log.Printf("Creating connection with %s (%s)", address, host)
				color.Unset()
				go eventinfrastructure.SendConnectionRequest("http://"+address, cr, false)
			}
			return
		}
	}
}

func GetStatus(context echo.Context) error {
	var s statusinfrastructure.Status
	var err error
	s.Version, err = statusinfrastructure.GetVersion("version.txt")
	if err != nil {
		s.Version = "missing"
		s.Status = statusinfrastructure.StatusSick
		s.StatusInfo = fmt.Sprintf("Error: %s", err.Error())
	} else {
		s.Status = statusinfrastructure.StatusOK
		s.StatusInfo = ""
	}

	return context.JSON(http.StatusOK, s)
}
