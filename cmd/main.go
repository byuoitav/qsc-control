package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/byuoitav/qsc-control/device"
	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
)

var slogger *slog.Logger

func main() {
	var port, logLevel string
	pflag.StringVarP(&port, "port", "p", "8016", "port on which to host the control service")
	pflag.StringVarP(&logLevel, "log", "l", "Info", "initial log level")
	pflag.Parse()

	//setup logger
	var slogLevel = new(slog.LevelVar)
	slogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))
	slog.SetDefault(slogger)

	// set log levels
	slogLevel.Set(slog.LevelInfo)

	if runtime.GOOS == "windows" {
		logLevel = "debug"
		slogLevel.Set(slog.LevelDebug)
		slogger.Info("running from Windows, logging set to debug")
	}

	port = ":" + port

	log, logLvl := buildLogger(logLevel)
	manager := device.DeviceManager{
		Log:      log,
		LogLevel: logLvl,
		DspList:  &sync.Map{},
	}

	router := gin.Default()

	router.Use(cors.Default())

	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "healthy")
	})

	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "healthy")
	})

	router.GET("/status", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "ok")
	})

	router.PUT("/log-level/:level", func(ctx *gin.Context) {
		lvl := ctx.Param("level")

		level, err := getZapLevelFromString(lvl)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, "invalid log level")
			return
		}

		err = setLogLevel(ctx.Param("level"), slogLevel)
		if err != nil {
			slogger.Error("can not set log level", "error", err)
			ctx.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		manager.LogLevel.SetLevel(level)
		ctx.JSON(http.StatusOK, gin.H{
			"current logLevel": slogLevel.Level(),
		})
	})

	router.GET("/log-level", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, manager.Log.Level().String())
	})

	err := manager.RunHTTPServer(router, port)
	if err != nil {
		manager.Log.Panic("http server failed")
	}
}

func setLogLevel(level string, logLevel *slog.LevelVar) error {
	level = strings.ToLower(level)
	if level == "debug" {
		logLevel.Set(slog.LevelDebug)
	} else if level == "info" {
		logLevel.Set(slog.LevelInfo)
	} else if level == "warn" {
		logLevel.Set(slog.LevelWarn)
	} else if level == "error" {
		logLevel.Set(slog.LevelError)
	} else {
		return fmt.Errorf("the debug level must be one of (debug, info, warn, error) received %s", level)
	}
	return nil
}
