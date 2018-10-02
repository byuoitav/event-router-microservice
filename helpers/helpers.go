package helpers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/byuoitav/common/db"
	"github.com/byuoitav/device-monitoring-microservice/statusinfrastructure"
	"github.com/byuoitav/messenger"
	"github.com/fatih/color"
	"github.com/labstack/echo"
)

var dev sync.Once

func SetMessageLogLevel(route *messenger.Router, context echo.Context) error {
	val := context.Param("val")
	if strings.ToLower(val) == "true" {

		route.SetMessageLogs(true)
		return context.JSON(http.StatusOK, "ok")

	} else if strings.ToLower(val) == "false" {
		route.SetMessageLogs(false)
		return context.JSON(http.StatusOK, "ok")

	}

	return context.JSON(http.StatusBadRequest, "Invalid value, must be true or false:")
}

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

func GetStatus(context echo.Context, route *messenger.Router) error {

	s := make(map[string]interface{})
	var err error
	s["version"], err = statusinfrastructure.GetVersion("version.txt")
	if err != nil {
		s["version"] = "missing"
		s["statuscode"] = statusinfrastructure.StatusSick
		s["statusinfo"] = fmt.Sprintf("Error: %s", err.Error())
	} else {
		s["statuscode"] = statusinfrastructure.StatusOK
		s["statusinfo"] = route.GetInfo()
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

	roomID := fmt.Sprintf("%s-%s", values[0], values[1])

	for {
		//first we need to make sure that the databse is ready for us.
		state, err := db.GetDB().GetStatus()
		if (err != nil || state != "completed") && (!(len(os.Getenv("DEV_ROUTER")) > 0) || !(len(os.Getenv("STOP_REPLICATION")) > 0)) {
			log.Printf(color.RedString("Database replication in state %v. Retrying in 5 seconds.", state))
			time.Sleep(5 * time.Second)
			continue
		}
		log.Printf(color.GreenString("Database replication state: %v", state))

		devices, err := db.GetDB().GetDevicesByRoomAndRole(roomID, "EventRouter")
		if err != nil {
			log.Printf(color.RedString("Connecting to the Configuration DB failed, retrying in 5 seconds."))
			time.Sleep(5 * time.Second)
			continue
		}

		if len(devices) == 0 {
			//there's a chance that there ARE routers in the room, but the initial database replication is occuring.
			//we're good, keep going
			state, err := db.GetDB().GetStatus()
			if err != nil || state != "completed" {
				log.Printf(color.RedString("Database replication in state %v. Retrying in 5 seconds.", state))
				time.Sleep(5 * time.Second)
				continue
			}
		}

		log.Printf(color.BlueString("Connection to the Configuration DB established."))

		regexStr := `[a-zA-z]+(\d+)$`
		re := regexp.MustCompile(regexStr)
		matches := re.FindAllStringSubmatch(pihn, -1)
		if len(matches) != 1 {
			log.Printf(color.RedString("Event router limited to only Control Processors."))
			return []string{}
		}

		mynum, err := strconv.Atoi(matches[0][1])
		log.Printf(color.YellowString("My processor Number: %v", mynum))

		for _, device := range devices {
			if len(os.Getenv("DEV_ROUTER")) > 0 {
				addresses = append(addresses, device.Address+":7000")
				dev.Do(func() {
					log.Printf(color.HiYellowString("Development device. Adding all event routers..."))
				})
				continue
			}

			//check if he's me
			if strings.EqualFold(device.ID, pihn) {
				continue
			}
			log.Printf(color.YellowString("Considering device: %v", device.Name))

			matches = re.FindAllStringSubmatch(device.Name, -1)
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

			addresses = append(addresses, device.Address+":7000")
		}
		break
	}
	log.Printf(color.HiGreenString("Done. Found %v routers", len(addresses)))
	return addresses
}
