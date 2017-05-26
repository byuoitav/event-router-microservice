package handlers

import (
	"net/http"

	"github.com/byuoitav/event-router-microservice/subscription"
	"github.com/labstack/echo"
)

func Subscribe(context echo.Context) error {
	var req subscription.SubscribeRequest
	err := context.Bind(&req)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, err.Error())
	}

	err = subscription.Subscribe(req)
	if err != nil {
		return context.JSON(http.StatusBadRequest, err.Error())
	}

	return context.JSON(http.StatusOK, context)
}
