package device

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/byuoitav/common/status"
	"github.com/byuoitav/connpool"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (dm *DeviceManager) HandlerGetVolume(ctx *gin.Context) {
	addr := ctx.Param("address")
	name := ctx.Param("name")
	name += "Gain"
	dsp := dm.CreateDSP(addr)

	dm.Log.Debug("getting volumes", zap.String("address", addr))

	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	vols, err := dsp.Volumes(c, []string{name})
	if err != nil {
		dm.Log.Error("unable to get volumes", zap.Error(err))
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	dm.Log.Debug("Got volumes", zap.String("address", addr), zap.String("volumes", fmt.Sprintf("%+v", vols)))

	vol, ok := vols[name]
	if !ok {
		dm.Log.Error("invalid name requested", zap.String("name", name))
		ctx.String(http.StatusBadRequest, "invalid name")
		return
	}

	ctx.JSON(http.StatusOK, status.Volume{
		Volume: vol,
	})
}

func (dm *DeviceManager) HandlerSetVolume(ctx *gin.Context) {
	addr := ctx.Param("address")
	name := ctx.Param("name")
	name += "Gain"
	dsp := dm.CreateDSP(addr)

	vol, err := strconv.Atoi(ctx.Param("volume"))
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	dm.Log.Debug("setting volume", zap.String("name", name), zap.Int("volume", vol))

	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	err = dsp.SetVolume(c, name, vol)
	if err != nil {
		dm.Log.Error("unable to set volume", zap.Error(err))
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	dm.Log.Debug("Set volume", zap.String("address", addr))
	ctx.JSON(http.StatusOK, status.Volume{
		Volume: vol,
	})
}

func (d *DSP) Volumes(ctx context.Context, blocks []string) (map[string]int, error) {
	toReturn := make(map[string]int)

	for i, block := range blocks {
		req := d.GetGenericGetStatusRequest(ctx)
		req.Params = append(req.Params, block)

		qscResp := QSCGetStatusResponse{}

		toSend, err := json.Marshal(req)
		if err != nil {
			return toReturn, err
		}
		toSend = append(toSend, 0x00)

		var resp []byte

		err = d.pool.Do(ctx, func(conn connpool.Conn) error {
			d.log.Info("Getting volume on %v", zap.String("block", block))
			conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

			n, err := conn.Write(toSend)
			switch {
			case err != nil:
				return fmt.Errorf("unable to write command to get volume for block %v: %v", block, err)
			case n != len(toSend):
				return fmt.Errorf("unable to write command to get volume for block %v: wrote %v/%v bytes", block, n, len(toSend))
			}

			deadline, ok := ctx.Deadline()
			if !ok {
				return fmt.Errorf("no deadline set")
			}

			resp, err = conn.ReadUntil(byte('\x00'), deadline)
			if err != nil {
				return fmt.Errorf("unable to read response: %w", err)
			}

			d.log.Debug("Got response: %v", zap.Any("response", resp))
			fmt.Printf("resp: %s\n", resp)

			return nil
		})
		if err != nil {
			return toReturn, err
		}
		resp = bytes.Trim(resp, "\x00")
		err = json.Unmarshal(resp, &qscResp)
		if err != nil {
			d.log.Error(err.Error())
			return toReturn, fmt.Errorf("error unmarshaling response: %v", err)
		}

		d.log.Debug(fmt.Sprintf("[QSC-Communication] Response received: %+v\n", qscResp))

		//get the volume out of the dsp and run it through our equation to reverse it
		found := false
		for _, res := range qscResp.Result {
			if res.Name == block {
				toReturn[blocks[i]] = d.DbToVolumeLevel(ctx, res.Value)
				found = true
				break
			}
		}
		if found {
			continue
		}

		return toReturn, errors.New("[QSC-Communication] No value returned with the name matching the requested state")
	}

	return toReturn, nil
}

func (d *DSP) SetVolume(ctx context.Context, block string, volume int) error {

	d.log.Debug(fmt.Sprintf("got: %v", volume))
	req := d.GetGenericSetStatusRequest(ctx)
	req.Params.Name = block

	if volume == 0 {
		req.Params.Value = -100
	} else {
		//do the logarithmic magic
		req.Params.Value = d.VolToDb(ctx, volume)
	}
	d.log.Debug(fmt.Sprintf("sending: %v", req.Params.Value))

	toSend, err := json.Marshal(req)
	if err != nil {
		return err
	}
	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		d.log.Info("setting volume on %v to %v", zap.String("block", block), zap.Int("level", volume))

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to set volume: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to set volume: wrote %v/%v bytes", n, len(toSend))
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			return fmt.Errorf("no deadline set")
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response: %w", err)
		}

		d.log.Debug("Got response: %v", zap.Any("response", resp))

		return nil
	})
	if err != nil {
		return err
	}

	//we need to unmarshal our response, parse it for the value we care about, then role with it from there
	qscResp := QSCSetStatusResponse{}
	resp = bytes.Trim(resp, "\x00")
	err = json.Unmarshal(resp, &qscResp)
	if err != nil {
		d.log.Error(err.Error())
		return err
	}
	if qscResp.Result.Name != block {
		errmsg := fmt.Sprintf("Invalid response, the name recieved does not match the name sent %v/%v", block, qscResp.Result.Name)
		d.log.Error(errmsg)
		return errors.New(errmsg)
	}

	return nil
}

func (d *DSP) DbToVolumeLevel(ctx context.Context, level float64) int {
	return int(math.Pow(10, (level/20)) * 100)
}

func (d *DSP) VolToDb(ctx context.Context, level int) float64 {
	return math.Log10(float64(level)/100) * 20
}
