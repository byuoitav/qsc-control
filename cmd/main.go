package main

import (
	"net/http"

	"github.com/byuoitav/qsc-control/device"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
)

func main() {
	var port, logLevel string
	pflag.StringVarP(&port, "port", "p", "8016", "port on which to host the control service")
	pflag.StringVarP(&logLevel, "log", "l", "Info", "initial log level")
	pflag.Parse()

	port = ":" + port

	log, logLvl := buildLogger(logLevel)
	manager := device.DeviceManager{
		Log:      log,
		LogLevel: logLvl,
	}

	router := gin.Default()

	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "healthy")
	})

	router.GET("/status")

	router.PUT("/log-level/:level", func(ctx *gin.Context) {
		lvl := ctx.Param("level")

		level, err := getZapLevelFromString(lvl)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, "invalid log level")
			return
		}

		manager.LogLevel.SetLevel(level)
		ctx.String(http.StatusOK, lvl)
	})

	router.GET("/log-level", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, manager.Log.Level().String())
	})

	err := manager.RunHTTPServer(router, port)
	if err != nil {
		manager.Log.Panic("http server failed")
	}
}
