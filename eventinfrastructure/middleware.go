package eventinfrastructure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/labstack/echo"
)

const ContextPublisher = "publisher"
const ContextSubscriber = "subscriber"

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

func SendConnectionRequest(url string, req ConnectionRequest, retry bool) error {
	defer color.Unset()
	body, err := json.Marshal(req)
	if err != nil {
		color.Set(color.FgHiRed)
		log.Printf("[error] %s", err.Error())
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	for err != nil || resp.StatusCode != 200 {
		color.Set(color.FgHiRed)
		if resp != nil {
			body, _ := ioutil.ReadAll(resp.Body)
			log.Printf("[error] failed to post. Response: (%v) %s", resp.StatusCode, body)
		} else {
			log.Printf("[error] failed to post. Error: %s", err.Error())
		}

		if !retry {
			break
		}

		log.Printf("Trying again in 5 seconds.")
		time.Sleep(5 * time.Second)
		resp, err = http.Post(url, "application/json", bytes.NewBuffer(body))
	}

	if err == nil {
		if resp.StatusCode == 200 {
			color.Set(color.FgHiGreen)
			log.Printf("Successfully posted connection request to %s", url)
			resp.Body.Close()
			return nil
		}
	}
	return errors.New(fmt.Sprintf("failed to post connection request to %s", url))
}

func GetIP() string {
	defer color.Unset()
	var ip net.IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return err.Error()
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && strings.Contains(address.String(), "/24") {
			ip, _, err = net.ParseCIDR(address.String())
			if err != nil {
				color.Set(color.FgHiRed)
				log.Fatalf("[error] %s", err.Error())
			}
		}
	}

	if ip == nil {
		color.Set(color.FgRed)
		log.Printf("[error] failed to find an non-loopback IP Address. Using PI_HOSTNAME/DEVELOPMENT_HOSTNAME as IP.")

		devhn := os.Getenv("DEVELOPMENT_HOSTNAME")
		if len(devhn) != 0 {
			color.Set(color.FgYellow)
			log.Printf("Development machine. Using hostname %s", devhn)
			return devhn
		}

		pihn := os.Getenv("PI_HOSTNAME")
		if len(pihn) == 0 {
			color.Set(color.FgRed)
			log.Fatalf("[error] PI_HOSTNAME is not set.")
		}
		return pihn
	}

	color.Set(color.FgHiGreen, color.Bold)
	log.Printf("My IP address is %v", ip.String())
	return ip.String()
}
