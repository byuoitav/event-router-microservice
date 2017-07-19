package eventinfrastructure

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo"
)

const ContextPublisher = "publisher"
const ContextSubscriber = "subscriber"
const ContextFilters = "filters"
const ContextPublisherAddress = "publisheraddress"

func BindPublisherAndSubscriber(p *Publisher, s *Subscriber) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextPublisher, s)
			c.Set(ContextSubscriber, p)
			return next(c)
		}
	}
}

func BindPublisher(p *Publisher) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextPublisher, p)
			return next(c)
		}
	}
}

func BindSubscriber(s *Subscriber) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextSubscriber, s)
			return next(c)
		}
	}
}

func BindFiltersAndPublisherAddress(filters []string, publisherAddr string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextFilters, filters)
			c.Set(ContextPublisherAddress, publisherAddr)
			return next(c)
		}
	}
}

func Subscribe(context echo.Context) error {
	var cr ConnectionRequest
	context.Bind(&cr)
	log.Printf("Recieved subscription request for %s", cr.PublisherAddr)

	// this awful looking chain just makes sure that each of the parameters are set
	// and pulls them out of context
	s := context.Get(ContextSubscriber)
	if sub, ok := s.(*Subscriber); ok {
		f := context.Get(ContextFilters)
		if filters, ok := f.([]string); ok {
			p := context.Get(ContextPublisherAddress)
			if pubAddr, ok := p.(string); ok {
				err := sub.HandleConnectionRequest(cr, filters, pubAddr)
				if err != nil {
					return context.JSON(http.StatusInternalServerError, err.Error())
				}
			} else {
				return context.JSON(http.StatusInternalServerError, errors.New("Publisher Address is not set"))
			}
		} else {
			return context.JSON(http.StatusInternalServerError, errors.New("Filters are not set"))
		}
	} else {
		return context.JSON(http.StatusInternalServerError, errors.New("Subscriber is not set"))
	}

	return context.JSON(http.StatusOK, nil)
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
