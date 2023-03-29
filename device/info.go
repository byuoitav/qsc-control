package device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/byuoitav/connpool"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (dm *DeviceManager) HandlerGetInfo(ctx *gin.Context) {
	addr := ctx.Param("address")
	dsp := dm.CreateDSP(addr)

	dm.Log.Debug("getting qsc hardware info", zap.String("address", addr))

	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	info, err := dsp.Info(c)
	if err != nil {
		dm.Log.Error("unable to get hardware info", zap.String("address", addr), zap.Error(err))
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	dm.Log.Debug("Got hardware info", zap.String("address", addr), zap.String("info", fmt.Sprintf("%+v", info)))

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"Info": info,
	})
}

type Info struct {
	Hostname   string
	ModelName  string
	IPAddress  string
	State      string
	StatusCode string
	RawState   string
}

// Info is all the juicy details about the QSC that everyone is DYING to know about
func (d *DSP) Info(ctx context.Context) (interface{}, error) {

	// toReturn is the struct of Hardware info
	var details Info

	var addr string
	d.pool.Do(ctx, func(conn connpool.Conn) error {
		addr = strings.Split(conn.RemoteAddr().String(), ":")[0]
		return nil
	})

	// get the hostname
	hostname, e := net.LookupAddr(addr)
	if e != nil {
		details.Hostname = addr
	} else {
		details.Hostname = strings.Trim(hostname[0], ".")
	}

	resp, err := d.GetStatus(ctx)
	if err != nil {
		return details, fmt.Errorf("there was an error getting the status: %v", err)
	}

	d.log.Info("response", zap.Any("response", resp))
	details.ModelName = resp.Result.Platform
	details.State = resp.Result.State
	details.StatusCode = resp.Result.Status.String

	details.IPAddress = addr

	rawString, _ := json.Marshal(resp)
	details.RawState = string(rawString)

	return details, nil
}

// Healthy .
func (d *DSP) Healthy(ctx context.Context) error {
	_, err := d.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed health check: %s", err)
	}

	return nil
}

// GetStatus will be getting responses for us I hope...
func (d *DSP) GetStatus(ctx context.Context) (QSCStatusGetResponse, error) {
	req := d.GetGenericStatusGetRequest(ctx)

	d.log.Info("In GetStatus...")
	toReturn := QSCStatusGetResponse{}

	toSend, err := json.Marshal(req)
	if err != nil {
		return toReturn, err
	}

	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		d.log.Info("getting status")

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to get status: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to get status: wrote %v/%v bytes", n, len(toSend))
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			deadline = time.Now().Add(3 * time.Second)
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response: %w", err)
		}

		d.log.Debug("Got response: %v", zap.Any("response", resp))

		return nil
	})
	if err != nil {
		return toReturn, err
	}

	resp = bytes.Trim(resp, "\x00")

	err = json.Unmarshal(resp, &toReturn)
	if err != nil {
		d.log.Info(err.Error())
	}

	return toReturn, err
}
