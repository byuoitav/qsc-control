package device

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/byuoitav/connpool"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (dm *DeviceManager) HandlerMute(ctx *gin.Context) {}

func (dm *DeviceManager) HandlerUnMute(ctx *gin.Context) {}

func (dm *DeviceManager) HandlerMuteStatus(ctx *gin.Context) {}

func (d *DSP) Mutes(ctx context.Context, blocks []string) (map[string]bool, error) {
	toReturn := make(map[string]bool)

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
			d.log.Info("Getting mute on %v", zap.String("block", block))
			conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

			n, err := conn.Write(toSend)
			switch {
			case err != nil:
				return fmt.Errorf("unable to write command to get mute for block %v: %v", block, err)
			case n != len(toSend):
				return fmt.Errorf("unable to write command to get mute for block %v: wrote %v/%v bytes", block, n, len(toSend))
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
			return toReturn, err
		}

		resp = bytes.Trim(resp, "\x00")
		err = json.Unmarshal(resp, &qscResp)
		if err != nil {
			d.log.Error(err.Error())
			return toReturn, err
		}

		//get the volume out of the dsp and run it through our equation to reverse it
		found := false
		for _, res := range qscResp.Result {
			if res.Name == block {
				if res.Value == 1.0 {
					toReturn[blocks[i]] = true
					found = true
				}
				if res.Value == 0.0 {
					toReturn[blocks[i]] = false
					found = true
				}
				break
			}
		}
		if found {
			continue
		}

		errmsg := "[QSC-Communication] No value returned with the name matching the requested state"
		d.log.Error(errmsg)
		return toReturn, errors.New(errmsg)
	}

	return toReturn, nil
}

func (d *DSP) SetMute(ctx context.Context, block string, mute bool) error {

	//we generate our set status request, then we ship it out

	req := d.GetGenericSetStatusRequest(ctx)
	req.Params.Name = block
	if mute {
		req.Params.Value = 1
	} else {
		req.Params.Value = 0
	}

	toSend, err := json.Marshal(req)
	if err != nil {
		return err
	}
	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		d.log.Info("setting mute on %v to %v", zap.String("block", block), zap.Bool("mute", mute))

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to set mute: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to set mute: wrote %v/%v bytes", n, len(toSend))
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

	//otherwise we check to see what the value is set to
	if qscResp.Result.Name != block {
		errmsg := fmt.Sprintf("Invalid response, the name recieved does not match the name sent %v/%v", block, qscResp.Result.Name)
		d.log.Error(errmsg)
		return errors.New(errmsg)
	}

	if qscResp.Result.Value == 1.0 {
		return nil
	}
	if qscResp.Result.Value == 0.0 {
		return nil
	}
	errmsg := fmt.Sprintf("[QSC-Communication] Invalid response received: %v", qscResp.Result)
	d.log.Error(errmsg)
	return errors.New(errmsg)
}
