package device

import (
	"github.com/gin-gonic/gin"
)

func (dm *DeviceManager) HandlerGetVolume(ctx *gin.Context) {
	// addr := ctx.Param("address")
	// name := ctx.Param("name")
	// name += "Gain"
	// dm.Log.Debug("getting volumes", zap.String("name", name))

}

func (dm *DeviceManager) HandlerSetVolume(ctx *gin.Context) {}
