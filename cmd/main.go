package main

import (
	"net/http"
	"qsc-control/device"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
)

func main() {
	var port, logLevel string
	pflag.StringVarP(&port, "port", "p", "8016", "port on which to host the control service")
	pflag.StringVarP(&logLevel, "log", "l", "Info", "initial log level")
	pflag.Parse()

	port = ":" + port

	manager := device.DeviceManager{
		Log: buildLogger(logLevel),
	}

	router := gin.Default()

	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "healthy")
	})

	router.GET("/status")
	router.PUT("/log-level/:level")
	router.GET("/log-level")

	err := manager.RunHTTPServer(router, port)
	if err != nil {
		manager.Log.Panic("http server failed")
	}
}
