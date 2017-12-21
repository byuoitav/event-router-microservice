package eventinfrastructure

import (
	"log"

	"github.com/byuoitav/event-router-microservice/base/router"
	"github.com/fatih/color"
)

func NewRouter(routingTable map[string][]string, addrs []string) (*router.Router, error) {

	r := router.NewRouter()

	err := r.StartRouter(routingTable)
	if err != nil {
		log.Printf(color.HiRedString("Could not start router: %v", err.Error()))
		return r, err
	}

	err = r.ConnectToRouters(addrs)
	if err != nil {
		log.Printf(color.HiRedString("Could not connect to peers: %v", err.Error()))
		return r, err
	}

	return r, nil
}
