package helpers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/byuoitav/av-api/dbo"
	"github.com/byuoitav/device-monitoring-microservice/statusinfrastructure"
	"github.com/byuoitav/event-router-microservice/base/router"
	"github.com/fatih/color"
	"github.com/labstack/echo"
)

func PrettyPrint(table map[string][]string) {

	color.Set(color.FgHiWhite)

	log.Printf("Printing Routing Table...")

	for k, v := range table {
		log.Printf("%v --> ", k)
		for _, val := range v {
			log.Printf("\t\t\t%v", val)
		}
	}
	log.Printf("Done.")
	color.Unset()
}

func GetStatus(context echo.Context, route *router.Router) error {

	var s statusinfrastructure.Status
	var err error
	s.Version, err = statusinfrastructure.GetVersion("version.txt")
	if err != nil {
		s.Version = "missing"
		s.Status = statusinfrastructure.StatusSick
		s.StatusInfo = fmt.Sprintf("Error: %s", err.Error())
	} else {
		s.Status = statusinfrastructure.StatusOK
		s.StatusInfo = route.GetInfo()
	}

	return context.JSON(http.StatusOK, s)
}

func GetOutsideAddresses() []string {
	log.Printf(color.HiGreenString("Getting all routers in the room..."))

	pihn := os.Getenv("PI_HOSTNAME")
	if len(pihn) == 0 {
		log.Fatalf("PI_HOSTNAME is not set.")
	}

	values := strings.Split(strings.TrimSpace(pihn), "-")
	addresses := []string{}

	for {
		devices, err := dbo.GetDevicesByBuildingAndRoomAndRole(values[0], values[1], "EventRouter")
		if err != nil {
			log.Printf(color.RedString("Connecting to the Configuration DB failed, retrying in 5 seconds."))
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf(color.BlueString("Connection to the Configuration DB established."))

		regexStr := `-CP(\d+)$`
		re := regexp.MustCompile(regexStr)
		matches := re.FindAllStringSubmatch(pihn, -1)
		if len(matches) != 1 {
			log.Printf(color.RedString("Event router limited to only Control Processors."))
			return []string{}
		}

		mynum, err := strconv.Atoi(matches[0][1])
		log.Printf(color.YellowString("My processor Number: %v", mynum))

		for _, device := range devices {

			//check if he's me
			if strings.EqualFold(device.GetFullName(), pihn) {
				continue
			}

			matches = re.FindAllStringSubmatch(device.Address, -1)
			if len(matches) != 1 {
				continue
			}

			num, err := strconv.Atoi(matches[0][1])
			if err != nil {
				continue
			}

			if num < mynum {
				continue
			}

			addresses = append(addresses, device.Address)
		}
		break
	}
	log.Printf(color.HiGreenString("Done. Found %v routers", len(addresses)))
	return addresses
}
