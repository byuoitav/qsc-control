package device

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DeviceManager struct {
	Log *zap.Logger
}

func (d *DeviceManager) RunHTTPServer(router *gin.Engine, port string) error {
	d.Log.Info("registering http endpoints")

	dev := router.Group("")
	dev.GET("/:address/:name/volume/mute", d.HandlerMute)
	dev.GET("/:address/:name/volume/unmute", d.HandlerUnMute)
	dev.GET("/:address/:name/mute/status", d.HandlerMuteStatus)
	dev.GET("/:address/:name/volume/set/:level", d.HandlerSetVolume)
	dev.GET("/:address/:name/volume/level", d.HandlerGetVolume)
	dev.PUT("/:address/generic/:name/:value", d.HandlerSetGeneric)
	dev.GET("/:address/generic/:name", d.HandlerGetGeneric)
	dev.GET("/:address/hardware", d.HandlerGetInfo)

	server := &http.Server{
		Addr:           port,
		MaxHeaderBytes: 1024 * 10,
	}

	d.Log.Info("running http server", zap.String("port", port))
	return router.Run(server.Addr)
}
