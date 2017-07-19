package eventinfrastructure

import "github.com/labstack/echo"

const ContextPublisher = "publisher"
const ContextSubscriber = "subscriber"

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
