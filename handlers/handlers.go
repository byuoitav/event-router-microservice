package handlers

import (
	"log"
	"net/http"

	"github.com/labstack/echo"
)

func Subscribe(context echo.Context) error {
	log.Printf("called subscribe")
	log.Printf("%s", context)

	return context.JSON(http.StatusOK, context)
}
